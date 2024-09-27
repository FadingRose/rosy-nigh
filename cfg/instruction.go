package cfg

import (
	"fadingrose/rosy-nigh/core/vm"
	"fmt"

	"github.com/holiman/uint256"
)

type instruction struct {
	pc   uint64
	arg  []byte
	op   vm.OpCode
	live bool

	JMPIBranch // support JUMPI / JUMP
	JMPBranch

	sloadVisit  int
	sstoreVisit int
}

func (i *instruction) Op() vm.OpCode {
	return i.op
}

func (i *instruction) PC() uint64 {
	return i.pc
}

func (i *instruction) Value() *uint256.Int {
	return uint256.NewInt(0).SetBytes(i.arg)
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
		argStr = fmt.Sprintf("0x%x", i.arg)
	}
	coverStr := ""
	slotCoverStr := ""
	switch i.op {
	case vm.JUMP:
		coverStr = i.JMPBranch.String()
	case vm.JUMPI:
		coverStr = i.JMPIBranch.String()
	}

	if i.op == vm.SLOAD || i.op == vm.SSTORE {
		slotCoverStr = "[ ]"
		if i.sloadVisit > 0 || i.sstoreVisit > 0 {
			slotCoverStr = "[x]"
		}
	}

	return fmt.Sprintf("0x%x(%d) %s %s %s%s", i.pc, i.pc, i.op.String(), argStr, coverStr, slotCoverStr)
}
