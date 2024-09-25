package vm

import (
	"errors"
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"strings"

	"github.com/holiman/uint256"
)

type Reg struct {
	// NOTE: We index a unique reg by depth, pc, loop
	// regkey       RegKey
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

	// for MLOAD | CallDataLoad
	offset uint64

	// for SLOAD | SSTORE
	slotkey   uint256.Int
	slotvalue uint256.Int

	ArgIndex *abi.ArgIndex
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

func (r *Reg) Name() string {
	return r.name()
}

func (r *Reg) name() string {
	return fmt.Sprintf("Reg#[%d,%d,%d]", r.index[0], r.index[1], r.index[2])
}

func (r *Reg) pc() uint64 {
	return r.index[1]
}

func (r *Reg) OpCode() OpCode {
	return r.op
}

func (r *Reg) RegKey() RegKey {
	return RegKey{
		index: r.index,
		reg:   r,
	}
}

func (r *Reg) Expand() string {
	return r.expand(0)
}

func (r *Reg) IsMagic() bool {
	mgops := []OpCode{
		CALLVALUE,
	}

	for _, op := range mgops {
		if r.op == op {
			return true
		}
	}
	return false
}

// IsBarrier returns true if the reg is a barrier, which means the reg is READ from memory/slot/tslot
func (r *Reg) IsBarrier() bool {
	bops := []OpCode{
		MLOAD,
		SLOAD,
		TLOAD,
	}
	for _, op := range bops {
		if r.op == op {
			return true
		}
	}
	return false
}

func (r *Reg) expand(depth int) string {
	if depth > 1024 {
		log.Warn("reg expand overflow, depth > 1024")
		return ""
	}
	var builder strings.Builder
	// indentation for the tree structure
	indent := strings.Repeat(".", 6)
	indents := strings.Repeat(indent, depth)

	nameWithOp := func(r *Reg) string {
		return fmt.Sprintf("%s %s", r.name(), r.op.String())
	}

	// isSkip returns true if we do NOT expand the next node
	isSkip := func(op OpCode) bool {
		control := []OpCode{STOP, POP, JUMPDEST}
		for _, c := range control {
			if c == op {
				return true
			}
		}
		if op.IsDup() || op.IsLog() || op.IsSwap() || op.IsPush() {
			return true
		}
		return false
	}

	val := r.Data.Hex()

	if depth == 0 {
		builder.WriteString(fmt.Sprintf("%s\n", nameWithOp(r)))
	} else {
		// Write the current node
		builder.WriteString(fmt.Sprintf("%s   └── %s <- %s\n", indents, nameWithOp(r), val))
	}

	if isSkip(r.op) {
		return builder.String()
	}

	for _, it := range r.itor() {
		if it == r.me {
			continue
		}
		if func(a, b [3]uint64) bool {
			return a[0] == b[0] && a[1] == b[1] && a[2] == b[2]
		}(it.index, r.index) {
			continue
		}
		builder.WriteString(it.expand(depth + 1))
	}

	return builder.String()
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

	if r.op == MLOAD {
		r.offset = params[0].reg.Data.Uint64()
	}

	if r.op == CALLDATALOAD {
		// skip function selector
		if params[0].reg.Data.Uint64() >= uint64(0x04) {
			r.offset = params[0].reg.Data.Uint64()
		}
	}

	if r.op == SLOAD {
		r.slotkey = params[0].reg.Data
		r.slotvalue = r.Data
	}

	if r.op == SSTORE {
		r.slotvalue = params[0].reg.Data
		r.slotkey = params[1].reg.Data
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

// Relies recursively return all the regkeys that this reg relies on
func (r *Reg) Relies() []RegKey {
	ret := make([]RegKey, 0)
	relies := func(reg *Reg) {
		if reg == nil {
			return
		}
		if reg == r || reg.me == r {
			ret = append(ret, reg.RegKey())
			return
		}
		ret = append(ret, reg.Relies()...)
	}

	for _, it := range r.itor() {
		if it == nil {
			break
		}
		relies(it)
	}

	return ret
}

func (r *Reg) itor() []*Reg {
	var ret []*Reg

	if r.cp == nil {
		ret = []*Reg{r.me, r.L, r.M, r.R0, r.R1, r.R2, r.R3, r.R4}
	} else {
		ret = []*Reg{r.cp, r.L, r.M, r.R0, r.R1, r.R2, r.R3, r.R4}
	}

	for i := range ret {
		if ret[i] == nil {
			ret = ret[:i]
			break
		}
	}
	return ret
}

func (r *Reg) GetBindName() (string, error) {
	if r.ArgIndex != nil {
		return r.ArgIndex.BindName(), nil
	}

	if r.cp != nil {
		return r.cp.GetBindName()
	}

	return "", errors.New("reg ont bound with any input")
}

func (r *Reg) GetBindType() (string, error) {
	if r.ArgIndex != nil {
		return r.ArgIndex.BindType(), nil
	}

	if r.cp != nil {
		return r.cp.GetBindType()
	}

	return "", errors.New("reg ont bound with any input")
}
