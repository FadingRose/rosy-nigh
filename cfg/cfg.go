package cfg

import (
	"fadingrose/rosy-nigh/cfg/symbolic"
	"fadingrose/rosy-nigh/core/asm"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type CFG struct {
	Blocks   []*Block // blocks[0] is entry; order otherwise undefined
	PathDict *PathDict

	blockMap        map[uint64]*Block
	directSuccessor map[uint64]*Block

	slotCoverage SlotCoverage

	accessList map[string][]SlotAccess
	rwmap      *RWMap
}

func NewCFG(bytecode []byte) *CFG {
	var (
		blocks      []*Block
		sstoreTotal = 0
		sloadTotal  = 0
	)
	cur := newBlock(0)
	cur.Selector = true // start collect statements from root

	blockmap := make(map[uint64]*Block)
	directSuccessor := make(map[uint64]*Block)
	it := asm.NewInstructionIterator(bytecode)
	nxtit := asm.NewInstructionIterator(bytecode)
	nxtit.Next()

	for it.Next() {
		cur.appendStmt(&instruction{
			pc:         it.PC(),
			arg:        it.Arg(),
			op:         it.Op(),
			live:       false,
			JMPBranch:  make(JMPBranch),
			JMPIBranch: make(JMPIBranch),
		})

		if it.Op() == vm.SSTORE {
			sstoreTotal++
		}

		if it.Op() == vm.SLOAD {
			sloadTotal++
		}
		nxtit.Next()
		if isLeadingEnd(it.Op(), nxtit.Op()) {
			idx := cur.Index + 1
			blocks = append(blocks, cur)
			cur = newBlock(idx)
		}

		blockmap[it.PC()] = cur
	}

	for i := 0; i < len(blocks)-1; i++ {
		ls := blocks[i].LastStmt()
		if ls.op != vm.JUMPI {
			continue
		}
		// HACK:
		fmt.Printf("ls.pc %d -> ds.pc %d\n", ls.pc, blocks[i+1].FirstStmt().pc)
		directSuccessor[ls.pc] = blocks[i+1]
	}

	return &CFG{
		Blocks:   blocks,
		PathDict: NewPathDict(),

		blockMap:        blockmap,
		directSuccessor: directSuccessor,
		slotCoverage: SlotCoverage{
			0, sstoreTotal, 0, sloadTotal,
		},
		accessList: make(map[string][]SlotAccess),
		rwmap:      nil,
	}
}

func (cfg *CFG) BlockMap(pc uint64) *Block {
	return cfg.blockMap[pc]
}

func (cfg *CFG) String() string {
	var (
		blocks   string
		coverage string
	)

	// coveredBranch, totalBranch := cfg.BranchCoverage()
	// coveredStmt, totalStmt := cfg.StatementCoverage()
	// coverage = fmt.Sprintf("Branch coverage: %d/%d  Statement Coverage: %d/%d\n", coveredBranch, totalBranch, coveredStmt, totalStmt)
	coverage = cfg.CoverageString()
	for _, block := range cfg.Blocks {
		blocks += fmt.Sprintf("%s\n", block.String())
	}
	return fmt.Sprintf("CFG:\n%s%s", coverage, blocks)
}

func (cfg *CFG) CoverageString() string {
	coveredBranch, totalBranch := cfg.branchCoverage()
	coveredStmt, totalStmt := cfg.StatementCoverage()
	slotCoverage := cfg.SlotCoverage()
	return fmt.Sprintf("Branch coverage: %d/%d  Statement Coverage: %d/%d\n SlotCoverage: R(%d/%d) W(%d/%d)\n", coveredBranch, totalBranch, coveredStmt, totalStmt, slotCoverage.SLOADCover, slotCoverage.SLOADTotal, slotCoverage.SSTORECover, slotCoverage.SSTORETotal)
}

// Update updates the CFG with the given list of register keys, then returns slot access list.
func (cfg *CFG) Update(reglist []vm.RegKey, funcname string) (readlist []SlotAccess, writelist []SlotAccess) {
	for _, key := range reglist {
		var (
			op        = key.OpCode()
			pc        = key.PC()
			dest      = key.Dest()
			cond      = key.Cond()
			slotKey   = key.SlotKey()
			slotValue = key.SlotValue()
		)

		if op == vm.SLOAD || op == vm.SSTORE {
			cfg.accessList[funcname] = append(cfg.accessList[funcname], SlotAccess{
				AccessType: func() AccessType {
					if op == vm.SLOAD {
						return Read
					}
					return Write
				}(),
				Key:   slotKey,
				Value: slotValue,
			})
		}

		switch op {
		case vm.JUMP:
			cfg.visitJMP(pc, dest)
		case vm.JUMPI:
			cfg.visitJMPI(pc, dest, cond)
		default:
			cfg.visit(pc)
		}
	}
	return
}

func (cfg *CFG) StringCoverage() string {
	coveredBranch, totalBranch := cfg.branchCoverage()
	coveredStmt, totalStmt := cfg.StatementCoverage()
	return fmt.Sprintf("Branch coverage: %d/%d  Statement Coverage: %d/%d\n", coveredBranch, totalBranch, coveredStmt, totalStmt)
}

func (cfg *CFG) SlotCoverage() SlotCoverage {
	var (
		sstoreCover = 0
		sloadCover  = 0
	)

	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			if stmt.sstoreVisit > 0 {
				sstoreCover++
			}
			if stmt.sloadVisit > 0 {
				sloadCover++
			}
		}
	}
	cfg.slotCoverage.SLOADCover = sloadCover
	cfg.slotCoverage.SSTORECover = sstoreCover
	return cfg.slotCoverage
}

// StatementCoverage returns the number of statements covered and total potential statements.
func (cfg *CFG) StatementCoverage() (int, int) {
	var (
		covered = 0
		total   = 0
	)
	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			if stmt.live {
				covered++
			}
			total++
		}
	}
	return covered, total
}

func (cfg *CFG) BranchCoverageLine(pc uint64) (int, int) {
	stmt := cfg.pcToStmt(pc)
	return stmt.BranchCoverage()
}

func (cfg *CFG) branchCoverage() (int, int) {
	var (
		covered = 0
		total   = 0
	)
	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			branchCovered, branchTotal := stmt.BranchCoverage()
			covered += branchCovered
			total += branchTotal
		}
	}
	return covered, total
}

func (cfg *CFG) pcToStmt(pc uint64) *instruction {
	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			if stmt.pc == pc {
				return stmt
			}
		}
	}
	return nil
}

func (cfg *CFG) visit(pc uint64) {
	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			if stmt.pc == pc {
				if stmt.op == vm.SLOAD {
					stmt.sloadVisit++
				}
				if stmt.op == vm.SSTORE {
					stmt.sstoreVisit++
				}
				block.Live = true
				stmt.live = true
				break
			}
		}
	}
}

func (cfg *CFG) visitJMP(pc, dest uint64) {
	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			if stmt.pc == pc {
				block.Live = true
				stmt.live = true
				stmt.CoverJump(int32(dest))
				break
			}
		}
	}
}

func (cfg *CFG) visitJMPI(pc, dest, cond uint64) {
	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			if stmt.pc == pc {
				block.Live = true
				stmt.live = true
				if cond == 0 {
					stmt.CoverJumpI(int32(dest), FalseBranch)
				} else {
					stmt.CoverJumpI(int32(dest), TrueBranch)
				}
				break
			}
		}
	}
}

func (cfg *CFG) lastPC() uint64 {
	return cfg.Blocks[len(cfg.Blocks)-1].LastStmt().PC()
}

func (cfg *CFG) ExtractPath(reglist []vm.RegKey) *Path {
	from_pc := uint64(0)

	path := &Path{}

	start_pc := reglist[0].PC()
	if bb, exists := cfg.blockMap[start_pc]; exists {
		path.Start = bb
		path.Start_PC = start_pc
	}

	cur := start_pc

	for i, key := range reglist {
		var (
			op   = key.OpCode()
			pc   = key.PC()
			dest = key.Dest()
			cond = key.Cond()
		)
		if i == len(reglist)-1 {
			path.Terminate_PC = cur
			if bb, exists := cfg.blockMap[cur]; exists {
				path.Terminate = bb
			} else {
				path.Terminate = nil
			}
			break
		}
		// make sure the _to is not nil
		switch op {
		case vm.JUMP:
			// to_pc := reglist[i+1].PC()
			to_pc := dest

			if bb, exists := cfg.blockMap[from_pc]; exists {
				reason := JUMP
				path.AddCheckpoint(from_pc, bb, reason, to_pc, cfg.blockMap[pc])
			} else {
				reason := JUMPI_FALSE
				path.AddCheckpoint(from_pc, nil, reason, to_pc, nil)
			}

			cur = from_pc
			from_pc = to_pc

		case vm.JUMPI:
			// to_pc := reglist[i+1].PC()

			var reason JUMP_TYPE
			if cond == uint64(0) {
				// FALSE - BRANCH
				reason = JUMPI_FALSE
			} else {
				reason = JUMPI_TRUE
			}

			to_pc := dest

			if bb, exists := cfg.blockMap[from_pc]; exists {
				path.AddCheckpoint(from_pc, bb, reason, to_pc, cfg.blockMap[pc])
			} else {
				path.AddCheckpoint(from_pc, nil, reason, to_pc, nil)
			}
		default:
			continue
		}
	}

	log.Debug("Extracted Path: %s", path)
	return path
}

func (cfg *CFG) RWMap() *RWMap {
	cfg.rwmap = NewRWMap(cfg.accessList)
	return cfg.rwmap
}

func (cfg *CFG) AccessList() map[string][]SlotAccess {
	return cfg.accessList
}

func (cfg *CFG) SymbolicResolve() {
	var (
		inter *symbolic.SymbolicInterpreter
		lut   = make(map[uint64]*symbolic.Operation, 0)
		table = vm.NewVerkleInstructionSet()
	)

	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			op := stmt.op
			minst, maxst := table[op].Instance().StackSize()
			lut[stmt.PC()] = symbolic.NewOperation(
				minst,
				int(params.StackLimit)+minst-maxst,
				op,
				stmt.PC(),
				stmt.Value(),
				symbolic.Executor(op))
		}
	}

	pathes := cfg.entryPathes()
	inter = symbolic.NewSymbolicInterpreter(lut)

	sp := &safepathes{
		pathes:     pathes,
		mu:         sync.Mutex{},
		pathHashes: make(map[uint64]struct{}),
		index:      int32(0),
		maxdepth:   len(cfg.Blocks) + 1,
	}

	var wg sync.WaitGroup
	workerCount := runtime.GOMAXPROCS(0)
	// workerCount := 1

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				curPath, ok := sp.get()

				if !ok {
					return
				}

				stmts := func(path []uint64) []uint64 {
					var ret []uint64
					for _, pc := range path {
						block := cfg.BlockMap(pc)
						for _, stmt := range block.Stmt {
							ret = append(ret, stmt.PC())
						}
					}
					return ret
				}(curPath)

				if len(stmts) == 0 {
					continue
				}

				dest, halt, err := inter.Run(stmts)
				if err != nil {
					continue
				}

				// reached REVERT | RETURN | STOP | INVALID
				if halt {
					continue
				}

				if dest != nil {
					if destBlock := cfg.blockMap[dest.Uint64()]; destBlock != nil {
						destBlock.Discovered()
						np := append(curPath, destBlock.FirstStmt().PC())
						// fmt.Printf("cur: %v dt: %v\n", curPath, np)
						sp.append(np)
					}
				}

				if ds := cfg.directSuccessor[stmts[len(stmts)-1]]; ds != nil {
					ds.Discovered()
					np := append(curPath, ds.FirstStmt().PC())
					// fmt.Println("ds: ", np)
					// fmt.Printf("cur: %v ds: %v\n", curPath, np)
					sp.append(np)
				}

			}
		}()
	}

	wg.Wait()
	fmt.Println(sp.string())
	fmt.Println(cfg.String())
	os.Exit(1)
}

func (cfg *CFG) entryPathes() [][]uint64 {
	var (
		entries [][]uint64
		prefix  []uint64
	)

	destFromSelector := func(b *Block) *uint256.Int {
		for i, stmt := range b.Stmt {
			if stmt.op == vm.PUSH2 {
				if b.Stmt[i+1].op == vm.JUMPI || b.Stmt[i+1].op == vm.JUMP {
					b.Selector = true
					return stmt.Value()
				}
			}
			if stmt.op == vm.STOP {
				b.Selector = true
			}
		}
		return nil
	}

	for _, block := range cfg.Blocks {
		prefix = append(prefix, block.FirstStmt().PC())

		if len(prefix) == 1 {
			continue
		}

		path := make([]uint64, len(prefix))
		copy(path, prefix)

		if dest := destFromSelector(block); dest == nil {
			break
		} else {
			if nxt := cfg.BlockMap(dest.Uint64()); nxt == nil {
				log.Debug("function selector with a invalid jump destination", "dest", dest.Hex())
			} else {
				path = append(path, nxt.FirstStmt().PC())
				entries = append(entries, path)
			}
		}
	}

	return entries
}
