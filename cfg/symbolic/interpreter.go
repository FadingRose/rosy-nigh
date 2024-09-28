package symbolic

import (
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fmt"

	"github.com/holiman/uint256"
)

type SymbolicInterpreter struct {
	lut map[uint64]*operation
}

func NewSymbolicInterpreter(Lut map[uint64]*Operation) *SymbolicInterpreter {
	return &SymbolicInterpreter{
		lut: func(L map[uint64]*Operation) map[uint64]*operation {
			lut := make(map[uint64]*operation)
			for k, v := range L {
				lut[k] = v.instance()
			}
			return lut
		}(Lut),
	}
}

func (si *SymbolicInterpreter) Run(stmts []uint64) (dest *uint256.Int, halt bool, err error) {
	st := newStack()
	for i, pc := range stmts {
		if si.lut[pc] == nil {
			return nil, false, fmt.Errorf("lut[%d] is nil", pc)
		}
		var (
			opera = si.lut[pc]
			op    = opera.op
		)

		opera.val = opera.solve()

		// params := make([]*operation, 0)

		// // HACK: track 2076
		// if stmts[len(stmts)-1] == 2148 {
		// 	fmt.Printf("pc: %d op:%s st: %s", pc, op.String(), st.string())
		// }
		if op == vm.POP {
			st.pop()
			continue
		}
		if op.IsDup() {
			err := st.dupN(op.DupNum())
			if err != nil {
				return nil, false, err
			}
			// opera = st.peek()
			continue
			// params = append(params, st.peek())
		} else if op.IsSwap() {
			err := st.dupN(op.SwapNum())
			if err != nil {
				return nil, false, err
			}
			// opera = st.peek()
			continue
			// params = append(params, st.peek())
		} else {
			params := st.popN(opera.paramSize)
			opera.params = params
		}

		// opera.params = params

		if len(opera.params) != opera.paramSize {
			return nil, false, fmt.Errorf("param size not match, op: %s, paramSize: %d, len(params): %d", op.String(), opera.paramSize, len(opera.params))
		}

		if i+1 == len(stmts) {
			if op == vm.REVERT || op == vm.STOP || op == vm.RETURN || op == vm.INVALID {
				log.Debug(fmt.Sprintf("%s at %d\n", op.String(), opera.pc))
				return nil, true, nil
			}

			if op == vm.JUMPI {
				dest = opera.params[1].val
			}
			if op == vm.JUMP {
				dest = opera.params[0].val
			}

			if dest == nil {
				log.Warn(fmt.Sprintf("resolve failed at %s\n", opera.expand(0)))
				return nil, false, fmt.Errorf("end of statements unreachable")
			}

			// log.Warn(fmt.Sprintf("resolve success at %s\n", opera.expand(0)))
			return dest, false, nil
		}

		if op == vm.JUMP || op == vm.JUMPI || op.IsDup() || op.IsSwap() {
			continue
		}
		if opera.pushbackSize > 0 {
			st.pushN(opera, opera.pushbackSize)
		}

		// // HACK: track 2076
		// if stmts[len(stmts)-1] == 2148 {
		// 	fmt.Printf("-> st: %s\n", st.string())
		// }
	}

	return nil, false, fmt.Errorf("unhandle error")
}
