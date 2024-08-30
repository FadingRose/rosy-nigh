package vm

import "github.com/holiman/uint256"

type Reg struct {
	// NOTE: We index a unique reg by depth, pc, loop
	index [3]uint64

	interpreter  *EVMInterpreter
	scopeContext *ScopeContext
	operation    *operation

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

	// ATTENTION: DONOT use this pc index a reg, use regkey
	// This pc used by JUMP/JUMPI, it will moving
	pc *uint64

	Data uint256.Int
}

func newReg(pc *uint64, index [3]uint64, interpreter *EVMInterpreter, scope *ScopeContext, operation *operation) *Reg {
	var r Reg

	r.index = index
	r.pc = pc
	r.interpreter = interpreter
	r.scopeContext = scope

	init := func(param *Reg, offset int) {
		switch offset {
		case 0:
			r.L = param
		case 1:
			r.M = param
		case 2:
			r.R0 = param
		case 3:
			r.R1 = param
		case 4:
			r.R2 = param
		case 5:
			r.R3 = param
		case 6:
			r.R4 = param
		default:
			panic("init reg fails: invalid offset")
		}
	}

	// In interpreter, we make sure items in stack at least minStack
	for i := 0; i < operation.minStack; i++ {
		paramReg := scope.Stack.checkParam(i)
		init(paramReg, i)
	}

	r.me = &r

	return &r
}

func (r *Reg) Solve() {
	// WARNING: we handle pc* moving here instead of in the executor
	r.operation.execute(r.pc, r.interpreter, r.scopeContext)
}

func (r *Reg) execute() ([]byte, error) {
	return r.operation.execute(r.pc, r.interpreter, r.scopeContext)
}
