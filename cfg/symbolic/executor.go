package symbolic

import (
	"fadingrose/rosy-nigh/core/vm"

	"github.com/holiman/uint256"
)

func Executor(op vm.OpCode) func(me *operation) *uint256.Int {
	return executor(op)
}

func executor(op vm.OpCode) func(me *operation) *uint256.Int {
	if op.IsPush() {
		return func(me *operation) *uint256.Int {
			return me.val
		}
	}

	ret := uint256.NewInt(0)

	switch op {
	case vm.ADD:
		return func(me *operation) *uint256.Int {
			ret = ret.Add(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.MUL:
		return func(me *operation) *uint256.Int {
			ret = ret.Mul(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.SUB:
		return func(me *operation) *uint256.Int {
			ret = ret.Sub(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.DIV:
		return func(me *operation) *uint256.Int {
			ret = ret.Div(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.SDIV:
		return func(me *operation) *uint256.Int {
			ret = ret.SDiv(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.MOD:
		return func(me *operation) *uint256.Int {
			ret = ret.Mod(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.SMOD:
		return func(me *operation) *uint256.Int {
			ret = ret.SMod(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.ADDMOD:
		return func(me *operation) *uint256.Int {
			ret = ret.AddMod(me.params[2].val, me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.MULMOD:
		return func(me *operation) *uint256.Int {
			ret = ret.MulMod(me.params[2].val, me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.EXP:
		return func(me *operation) *uint256.Int {
			ret = ret.Exp(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.SIGNEXTEND:
		return func(me *operation) *uint256.Int {
			ret = ret.ExtendSign(me.params[0].val, me.params[1].val)
			return ret
		}
	default:
		return func(me *operation) *uint256.Int {
			return nil
		}
	}
}
