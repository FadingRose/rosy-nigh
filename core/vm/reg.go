package vm

import "github.com/holiman/uint256"

type Reg struct {
	// NOTE: We index a unique reg by depth, pc, loop
	index     [3]uint64
	paramSize int
	op        OpCode
	Data      uint256.Int
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
}

func newReg(index [3]uint64, op OpCode, paramSize int) *Reg {
	var r Reg
	r.index = index
	r.me = &r
	r.op = op

	return &r
}
