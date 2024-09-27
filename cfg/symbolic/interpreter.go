package symbolic

import (
	"fadingrose/rosy-nigh/cfg"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type symbolicInterpreter struct {
	cfg       *cfg.CFG
	st        *stack
	laststmts []uint64
	lut       map[uint64]*operation
	pathes    [][]uint64
}

func NewSymbolicInterpreter(cfg *cfg.CFG) *symbolicInterpreter {
	table := vm.NewVerkleInstructionSet()

	// stmts = make([]uint64, 0)
	lut := make(map[uint64]*operation)

	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			op := stmt.Op()
			minst, maxst := table[op].Instance().StackSize()
			operation := &operation{
				paramSize:    minst,
				pushbackSize: int(params.StackLimit) + minst - maxst,
				op:           op,
				pc:           stmt.PC(),
				val:          stmt.Value(),
				exec:         executor(op),
			}
			lut[stmt.PC()] = operation
		}
	}

	pathes := make([][]uint64, 0)

	for _, b := range cfg.Blocks {
		if b.Stmt[0].Op() == vm.JUMPDEST {
			continue
		}
		path := []uint64{b.Stmt[0].PC()}
		pathes = append(pathes, path)
	}

	for _, path := range pathes {
		fmt.Println(path)
	}

	return &symbolicInterpreter{
		cfg:       cfg,
		st:        newStack(),
		laststmts: make([]uint64, 0),
		lut:       lut,
		pathes:    pathes,
	}
}

func (si *symbolicInterpreter) resolveCFG() {
	isRelunctant := func(newPath []uint64) bool {
		n := len(si.pathes)
		return func(a, b []uint64) bool {
			if len(a) != len(b) {
				return false
			}
			for i := 0; i < len(a); i++ {
				if a[i] != b[i] {
					return false
				}
			}
			return true
		}(newPath, si.pathes[n-1])
	}

	for len(si.pathes) > 0 {

		si.laststmts = make([]uint64, 0)
		// fmt.Printf("pathes: %v\n", si.pathes)
		for _, i := range si.pathes[0] {
			block := si.cfg.BlockMap(i)
			if block == nil {
				continue
			}
			si.laststmts = append(si.laststmts, block.PCs()...)
		}
		var (
			curPath          = si.pathes[0]
			directSuccesorPC = si.laststmts[len(si.laststmts)-1] + 1
		)

		si.pathes = si.pathes[1:]

		dest, err := si.run()
		if err != nil {
			// fmt.Printf("%v failed, %v\n", curPath, err)
			continue
		}

		var nxtBlock *cfg.Block
		if dest == nil {
			// fmt.Println("dest is nil")
		} else {
			nxtBlock = si.cfg.BlockMap(dest.Uint64())
		}

		if nxtBlock == nil {
			// fmt.Println("no block found")
		} else {
			newPath := append(curPath, nxtBlock.Stmt[0].PC())
			if !isRelunctant(newPath) {
				// fmt.Printf("new path dest %v\n", newPath)
				si.pathes = append(si.pathes, newPath)
				nxtBlock.Alive()
			}
		}

		var directSuccesor *cfg.Block
		if directSuccesorPC > si.cfg.Blocks[len(si.cfg.Blocks)-1].LastStmt().PC() {
			directSuccesor = nil
		} else {
			directSuccesor = si.cfg.BlockMap(directSuccesorPC)
			newPath := append(curPath, directSuccesor.Stmt[0].PC())
			if !isRelunctant(newPath) {
				directSuccesor.Alive()
				si.pathes = append(si.pathes, newPath)
			}
		}

	}
	fmt.Println("End of resolveCFG")
	fmt.Printf("%s", si.cfg.String())
	os.Exit(0)
}

func (si *symbolicInterpreter) run() (dest *uint256.Int, err error) {
	si.st = newStack()
	for i, pc := range si.laststmts {
		if si.lut[pc] == nil {
			return nil, fmt.Errorf("lut[%d] is nil", pc)
		}
		var (
			opera = si.lut[pc]
			op    = opera.op
			st    = si.st
		)

		opera.val = opera.solve()

		params := make([]*operation, 0)

		if op.IsDup() {
			err := st.dupN(op.DupNum())
			if err != nil {
				return nil, err
			}
			params = append(params, st.peek())
		} else if op.IsSwap() {
			err := st.dupN(op.SwapNum())
			if err != nil {
				return nil, err
			}
			params = append(params, st.peek())
		} else {
			params = st.popN(opera.paramSize)
		}

		if len(params) != opera.paramSize {
			return nil, fmt.Errorf("param size not match")
		}

		opera.params = params

		if i+1 == len(si.laststmts) {
			if op == vm.REVERT || op == vm.STOP || op == vm.RETURN || op == vm.INVALID {
				log.Debug(fmt.Sprintf("%s at %d\n", op.String(), opera.pc))
				return nil, fmt.Errorf(op.String())
			}

			if op == vm.JUMPI {
				dest = opera.params[1].val
			}
			if op == vm.JUMP {
				dest = opera.params[0].val
			}

			if dest == nil {
				return nil, fmt.Errorf("end of statements unreachable")
			}

			log.Warn(fmt.Sprintf("resolve success at %s\n", opera.expand(0)))
			return dest, nil
		}

		if op == vm.JUMP || op == vm.JUMPI || op.IsDup() || op.IsSwap() {
			continue
		}

		if opera.pushbackSize > 0 {
			st.pushN(opera, opera.pushbackSize)
		}
	}

	return nil, fmt.Errorf("unhandle error")
}
