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
	case vm.LT:
		return func(me *operation) *uint256.Int {
			ret = me.params[1].val
			if ret.Lt(me.params[0].val) {
				ret.SetOne()
			} else {
				ret.Clear()
			}
			return ret
		}
	case vm.GT:
		return func(me *operation) *uint256.Int {
			ret = me.params[1].val
			if ret.Gt(me.params[0].val) {
				ret.SetOne()
			} else {
				ret.Clear()
			}
			return ret
		}
	case vm.SLT:
		return func(me *operation) *uint256.Int {
			ret = me.params[1].val
			if ret.Slt(me.params[0].val) {
				ret.SetOne()
			} else {
				ret.Clear()
			}
			return ret
		}
	case vm.SGT:
		return func(me *operation) *uint256.Int {
			ret = me.params[1].val
			if ret.Sgt(me.params[0].val) {
				ret.SetOne()
			} else {
				ret.Clear()
			}
			return ret
		}
	case vm.EQ:
		return func(me *operation) *uint256.Int {
			ret = me.params[1].val
			if ret.Eq(me.params[0].val) {
				ret.SetOne()
			} else {
				ret.Clear()
			}
			return ret
		}
	case vm.ISZERO:
		return func(me *operation) *uint256.Int {
			ret = me.params[0].val
			if ret.IsZero() {
				ret.SetOne()
			} else {
				ret.Clear()
			}
			return ret
		}
	case vm.AND:
		return func(me *operation) *uint256.Int {
			ret = ret.And(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.OR:
		return func(me *operation) *uint256.Int {
			ret = ret.Or(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.XOR:
		return func(me *operation) *uint256.Int {
			ret = ret.Xor(me.params[1].val, me.params[0].val)
			return ret
		}
	case vm.NOT:
		return func(me *operation) *uint256.Int {
			ret = ret.Not(me.params[0].val)
			return ret
		}
	case vm.BYTE:
		return func(me *operation) *uint256.Int {
			ret = me.params[0].val
			ret = ret.Byte(me.params[1].val)
			return ret
		}
	case vm.SHL:
		return func(me *operation) *uint256.Int {
			shift, value := me.params[1].val, me.params[0].val
			if shift.LtUint64(256) {
				ret.Lsh(value, uint(shift.Uint64()))
			} else {
				ret.Clear()
			}
			return ret
		}
	case vm.SHR:
		return func(me *operation) *uint256.Int {
			shift, value := me.params[1].val, me.params[0].val
			if shift.LtUint64(256) {
				ret.Rsh(value, uint(shift.Uint64()))
			} else {
				ret.Clear()
			}
			return ret
		}
	case vm.SAR:
		return func(me *operation) *uint256.Int {
			shift, value := me.params[1].val, me.params[0].val
			if shift.GtUint64(256) {
				if value.Sign() >= 0 {
					ret.Clear()
				} else {
					ret.SetAllOne()
				}
				return nil
			}
			ret.SRsh(value, uint(shift.Uint64()))
			return ret
		}
	default:
		return func(me *operation) *uint256.Int {
			return nil
		}
	}
}
