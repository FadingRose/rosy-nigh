package cfg

import (
	"fadingrose/rosy-nigh/core/asm"
	"fadingrose/rosy-nigh/core/vm"
	"fmt"
	"strings"
)

// For a JUMPI, whether it's FALSE or TRUE branch is coverd?
type JMPIBranch map[int32][2]bool

type JMPBranch map[int32]bool

type CondBranch = int

const (
	FalseBranch CondBranch = iota
	TrueBranch
)

func (jumpi JMPIBranch) String() string {
	var ret strings.Builder
	for index, branch := range jumpi {
		var (
			falseBranch = " "
			trueBranch  = " "
		)
		if branch[FalseBranch] {
			falseBranch = "x"
		}
		if branch[TrueBranch] {
			trueBranch = "x"
		}
		ret.WriteString(fmt.Sprintf("dest: 0x%x: false-branch:[%s] true-branch:[%s]\n", index, falseBranch, trueBranch))
	}
	return ret.String()
}

func (jump JMPBranch) String() string {
	var ret strings.Builder
	for index, covered := range jump {
		branch := " "
		if covered {
			branch = "x"
		}
		ret.WriteString(fmt.Sprintf("dest: 0x%x branch:[%s]\n", index, branch))
	}
	return ret.String()
}

type instruction struct {
	pc   uint64
	arg  []byte
	op   vm.OpCode
	live bool

	JMPIBranch // support JUMPI / JUMP
	JMPBranch
}

func (i *instruction) CoverJumpI(index int32, branch CondBranch) {
	if _, ok := i.JMPIBranch[index]; !ok {
		i.JMPIBranch[index] = [2]bool{false, false}
	}

	if branch == FalseBranch {
		i.JMPIBranch[index] = [2]bool{true, i.JMPIBranch[index][TrueBranch]}
	} else {
		i.JMPIBranch[index] = [2]bool{i.JMPIBranch[index][FalseBranch], true}
	}
}

func (i *instruction) CoverJump(index int32) {
	i.JMPBranch[index] = true
}

// BranchCoverage returns the number of branches covered and total potential branches.
func (i *instruction) BranchCoverage() (int, int) {
	var (
		covered = 0
		total   = 0
	)

	switch i.op {
	case vm.JUMP:
		total = len(i.JMPBranch)
		for _, branch := range i.JMPBranch {
			if branch {
				covered++
			}
		}
	case vm.JUMPI:
		total = len(i.JMPIBranch) * 2
		for _, branch := range i.JMPIBranch {
			if branch[FalseBranch] {
				covered++
			}
			if branch[TrueBranch] {
				covered++
			}
		}
	}

	return covered, total
}

func (i *instruction) String() string {
	argStr := ""
	if i.arg != nil {
		argStr = fmt.Sprintf("%x", i.arg)
	}
	coverStr := ""
	switch i.op {
	case vm.JUMP:
		coverStr = i.JMPBranch.String()
	case vm.JUMPI:
		coverStr = i.JMPIBranch.String()
	}
	return fmt.Sprintf("0x%x(%d) %s %s %s", i.pc, i.pc, i.op.String(), argStr, coverStr)
}

type Block struct {
	Succs []*Block
	Index int32
	Live  bool // block is reachable from entrypoint
	Stmt  []*instruction
}

func (b *Block) String() string {
	indent := ""
	live := ""
	if b.Live {
		indent = strings.Repeat(" ", 8)
		live = "(live)"
	}

	var stmts string
	for _, stmt := range b.Stmt {
		stmts += fmt.Sprintf("%s%s\n", indent, stmt.String())
	}

	return fmt.Sprintf("%sBlock %d:\n%s", live, b.Index, stmts)
}

func newBlock(index int32) *Block {
	return &Block{Succs: make([]*Block, 0), Index: index, Live: false, Stmt: make([]*instruction, 0)}
}

func (b *Block) appendStmt(stmt *instruction) {
	b.Stmt = append(b.Stmt, stmt)
}

// IsLeadingEnd returns true if the opcode is a leading end of a basic block.
// JUMP | JUMPI | STOP | RETURN | REVERT | INVALID | SELFDESTRUCT
func isLeadingEnd(op vm.OpCode) bool {
	switch op {
	case vm.JUMP, vm.JUMPI, vm.STOP, vm.RETURN, vm.REVERT, vm.INVALID, vm.SELFDESTRUCT:
		return true
	default:
		return false
	}
}

type CFG struct {
	Blocks []*Block // blocks[0] is entry; order otherwise undefined
}

func NewCFG(bytecode []byte) *CFG {
	var blocks []*Block

	cur := newBlock(0)
	cur.Live = true // start collect statements from root

	it := asm.NewInstructionIterator(bytecode)
	for it.Next() {
		cur.appendStmt(&instruction{
			pc:         it.PC(),
			arg:        it.Arg(),
			op:         it.Op(),
			live:       false,
			JMPBranch:  make(JMPBranch),
			JMPIBranch: make(JMPIBranch),
		})
		if isLeadingEnd(it.Op()) {
			idx := cur.Index + 1
			blocks = append(blocks, cur)
			cur = newBlock(idx)
		}
	}

	return &CFG{Blocks: blocks}
}

func (cfg *CFG) String() string {
	var (
		blocks   string
		coverage string
	)

	coveredBranch, totalBranch := cfg.BranchCoverage()
	coveredStmt, totalStmt := cfg.StatementCoverage()
	coverage = fmt.Sprintf("Branch coverage: %d/%d  Statement Coverage: %d/%d\n", coveredBranch, totalBranch, coveredStmt, totalStmt)
	for _, block := range cfg.Blocks {
		blocks += fmt.Sprintf("%s\n", block.String())
	}
	return fmt.Sprintf("CFG:\n%s%s", coverage, blocks)
}

func (cfg *CFG) Update(reglist []vm.RegKey) {
	for _, key := range reglist {
		var (
			op   = key.OpCode()
			pc   = key.PC()
			dest = key.Dest()
			cond = key.Cond()
		)
		switch op {
		case vm.JUMP:
			cfg.visitJMP(pc, dest)
		case vm.JUMPI:
			cfg.visitJMPI(pc, dest, cond)
		default:
			cfg.visit(pc)
		}
	}
}

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

func (cfg *CFG) BranchCoverage() (int, int) {
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

func (cfg *CFG) visit(pc uint64) {
	for _, block := range cfg.Blocks {
		for _, stmt := range block.Stmt {
			if stmt.pc == pc {
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
