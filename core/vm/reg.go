package vm

import (
	"fmt"
	"strings"

	"github.com/holiman/uint256"
)

type Reg struct {
	// NOTE: We index a unique reg by depth, pc, loop
	index        [3]uint64
	paramSize    int
	pushbackSize int
	op           OpCode
	Data         uint256.Int
	//[R4, R3, R2, R1, R0, M, L, me
	// stack parameters
	R4 *Reg `json:"r4"` //-7#stack
	R3 *Reg `json:"r3"` //-6#stack
	R2 *Reg `json:"r2"` //-5#stack
	R1 *Reg `json:"r1"` //-4#stack
	R0 *Reg `json:"R0"` //-3#stack
	M  *Reg `json:"M"`  //-2#stack
	L  *Reg `json:"L"`  //-1#stack
	me *Reg // 0#stack

	cp *Reg // swap / dup source reg

	// for JUMP / JUMPI
	dest uint64
	cond uint64
}

func newReg(index [3]uint64, op OpCode, paramSize int, pushbackSize int) *Reg {
	var r Reg
	r.index = index
	r.me = &r
	r.op = op
	r.paramSize = paramSize
	r.pushbackSize = pushbackSize
	return &r
}

func (r *Reg) name() string {
	return fmt.Sprintf("[%d,%d,%d]", r.index[0], r.index[1], r.index[2])
}

func (r *Reg) pc() uint64 {
	return r.index[1]
}

func (r *Reg) String() string {
	var builder strings.Builder

	if r.me == nil || r != r.me {
		builder.WriteString("!crash")
	}

	val := "<nil>"
	valdec := ""

	val = r.Data.Hex()
	valdec = "(" + r.Data.Dec() + ")"

	opcode := r.op.String()
	fixedLen := 8

	if len(opcode) < fixedLen {
		opcode = opcode + strings.Repeat(" ", fixedLen-len(opcode))
	}

	if r.cp != nil {
		builder.WriteString(fmt.Sprintf("%s->", r.cp.name()))
	}

	builder.WriteString(
		fmt.Sprintf(
			"%s %s PC: 0x%x(%d), Val: %s%s, params: %d, pushback: %d",
			r.name(),
			opcode,
			r.pc(),
			r.pc(),
			val,
			valdec,
			r.paramSize,
			r.pushbackSize,
		),
	)

	// if len(r.CrossTrack) > 0 {
	// 	builder.WriteString(", CrossTrack: ")
	// 	for _, ct := range r.CrossTrack {
	// 		builder.WriteString(fmt.Sprintf(" %s ", ct._name()))
	// 	}
	// }

	registers := []struct {
		name string
		reg  *Reg
	}{
		{
			"R4",
			r.R4,
		}, {"R3", r.R3}, {"R2", r.R2}, {"R1", r.R1}, {"R0", r.R0}, {"M", r.M}, {"L", r.L},
	}

	for _, regPair := range registers {
		if regPair.reg != nil {
			builder.WriteString(
				fmt.Sprintf(", %s: %s", regPair.name, regPair.reg.name()),
			)
		}
	}

	// builder.WriteString(r.Instruction.String())
	builder.WriteString("\n")

	return builder.String()
}

func (r *Reg) deepCopy() *Reg {
	copy := &Reg{
		index:        r.index,
		paramSize:    r.paramSize,
		pushbackSize: r.pushbackSize,
		op:           r.op,
		Data:         r.Data,
		R4:           r.R4,
		R3:           r.R3,
		R2:           r.R2,
		R1:           r.R1,
		R0:           r.R0,
		M:            r.M,
		L:            r.L,
	}
	copy.me = copy
	return copy
}

func (r *Reg) Duplicate() *Reg {
	dup := r.deepCopy()
	dup.cp = r // from `r` to `dup`
	return dup
}

func (r *Reg) setupParams(params []RegKey) {
	if r.op.IsDup() || r.op.IsSwap() {
		r.cp = params[0].reg
		return
	}

	if r.op == JUMP {
		r.dest = params[0].reg.Data.Uint64()
	}

	if r.op == JUMPI {
		r.cond = params[0].reg.Data.Uint64()
		r.dest = params[1].reg.Data.Uint64()
	}

	set := func(i int, reg *Reg) {
		switch i {
		case 0:
			r.L = reg
		case 1:
			r.M = reg
		case 2:
			r.R0 = reg
		case 3:
			r.R1 = reg
		case 4:
			r.R2 = reg
		case 5:
			r.R3 = reg
		case 6:
			r.R4 = reg
		default:
			panic("invalid index")
		}
	}
	for i, regKey := range params {
		set(i, regKey.reg)
	}
}
