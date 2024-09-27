package cfg

import (
	"fadingrose/rosy-nigh/core/vm"
	"fmt"
	"strings"
)

type Block struct {
	Succs    []*Block
	Index    int32
	Live     bool // block is reachable from entrypoint
	Discover bool // block is discovered in the CFG
	Selector bool // block is function Selector
	Stmt     []*instruction
}

func newBlock(index int32) *Block {
	return &Block{
		Succs:    make([]*Block, 0),
		Index:    index,
		Live:     false,
		Discover: false,
		Selector: false,
		Stmt:     make([]*instruction, 0),
	}
}

func (b *Block) FirstStmt() *instruction {
	return b.Stmt[0]
}

func (b *Block) LastStmt() *instruction {
	n := len(b.Stmt)
	return b.Stmt[n-1]
}

func (b *Block) PCs() []uint64 {
	var ret []uint64
	for _, stmt := range b.Stmt {
		ret = append(ret, stmt.pc)
	}
	return ret
}

func (b *Block) String() string {
	indent := ""
	live := ""
	discover := ""
	selector := ""

	if b.Discover || b.Live || b.Selector {
		indent = strings.Repeat(" ", 8)
	}

	if b.Selector {
		selector = "(selector)"
	} else {
		if b.Discover {
			discover = "(discover)"
		}
		if b.Live {
			live = "(live)"
		}
	}

	var stmts string
	for _, stmt := range b.Stmt {
		stmts += fmt.Sprintf("%s%s\n", indent, stmt.String())
	}

	return fmt.Sprintf("%s%s%sBlock %d:\n%s", selector, discover, live, b.Index, stmts)
}

func (b *Block) Alive() {
	b.Live = true
	for _, stmt := range b.Stmt {
		stmt.live = true
		if stmt.op == vm.SLOAD {
			stmt.sloadVisit++
		}
		if stmt.op == vm.SSTORE {
			stmt.sstoreVisit++
		}
	}
}

func (b *Block) Discovered() {
	b.Discover = true
}

func (b *Block) appendStmt(stmt *instruction) {
	b.Stmt = append(b.Stmt, stmt)
}

// IsLeadingEnd returns true if the opcode is a leading end of a basic block.
// JUMP | JUMPI | STOP | RETURN | REVERT | INVALID | SELFDESTRUCT
func isLeadingEnd(op vm.OpCode, nxtop vm.OpCode) bool {
	switch op {
	case vm.JUMP, vm.JUMPI, vm.STOP, vm.RETURN, vm.REVERT, vm.INVALID, vm.SELFDESTRUCT:
		return true
	default:
		return isLeadingStart(nxtop)
	}
}

// isLeadingStart returns true if the opcode is a leading start of a basic block.
// JUMPDEST
func isLeadingStart(op vm.OpCode) bool {
	return op == vm.JUMPDEST
}
