package symbolic

import (
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fmt"

	"github.com/holiman/uint256"
)

type SymbolicInterpreter struct {
	// cfg       *cfg.CFG
	// st  *stack
	lut map[uint64]*operation
}

func NewSymbolicInterpreter(Lut map[uint64]*Operation) *SymbolicInterpreter {
	// table := vm.NewVerkleInstructionSet()

	// stmts = make([]uint64, 0)
	// lut := make(map[uint64]*operation)

	// for _, block := range cfg.Blocks {
	// 	for _, stmt := range block.Stmt {
	// 		op := stmt.Op()
	// 		minst, maxst := table[op].Instance().StackSize()
	// 		operation := &operation{
	// 			paramSize:    minst,
	// 			pushbackSize: int(params.StackLimit) + minst - maxst,
	// 			op:           op,
	// 			pc:           stmt.PC(),
	// 			val:          stmt.Value(),
	// 			exec:         executor(op),
	// 		}
	// 		lut[stmt.PC()] = operation
	// 	}
	// }

	// pathes := make([][]uint64, 0)

	// for _, b := range cfg.Blocks {
	// 	if b.Stmt[0].Op() == vm.JUMPDEST {
	// 		continue
	// 	}
	// 	path := []uint64{b.Stmt[0].PC()}
	// 	pathes = append(pathes, path)
	// }

	return &SymbolicInterpreter{
		// cfg:       cfg,
		// st: newStack(),
		lut: func(L map[uint64]*Operation) map[uint64]*operation {
			lut := make(map[uint64]*operation)
			for k, v := range L {
				lut[k] = v.instance()
			}
			return lut
		}(Lut),
	}
}

// func (si *symbolicInterpreter) resolveCFG() {
// 	isRelunctant := func(newPath []uint64) bool {
// 		n := len(si.pathes)
// 		return func(a, b []uint64) bool {
// 			if len(a) != len(b) {
// 				return false
// 			}
// 			for i := 0; i < len(a); i++ {
// 				if a[i] != b[i] {
// 					return false
// 				}
// 			}
// 			return true
// 		}(newPath, si.pathes[n-1])
// 	}
//
// 	for len(si.pathes) > 0 {
//
// 		si.laststmts = make([]uint64, 0)
// 		// fmt.Printf("pathes: %v\n", si.pathes)
// 		for _, i := range si.pathes[0] {
// 			block := si.cfg.BlockMap(i)
// 			if block == nil {
// 				continue
// 			}
// 			si.laststmts = append(si.laststmts, block.PCs()...)
// 		}
// 		var (
// 			curPath          = si.pathes[0]
// 			directSuccesorPC = si.laststmts[len(si.laststmts)-1] + 1
// 		)
//
// 		si.pathes = si.pathes[1:]
//
// 		dest, err := si.run()
// 		if err != nil {
// 			// fmt.Printf("%v failed, %v\n", curPath, err)
// 			continue
// 		}
//
// 		var nxtBlock *cfg.Block
// 		if dest == nil {
// 			// fmt.Println("dest is nil")
// 		} else {
// 			nxtBlock = si.cfg.BlockMap(dest.Uint64())
// 		}
//
// 		if nxtBlock == nil {
// 			// fmt.Println("no block found")
// 		} else {
// 			newPath := append(curPath, nxtBlock.Stmt[0].PC())
// 			if !isRelunctant(newPath) {
// 				// fmt.Printf("new path dest %v\n", newPath)
// 				si.pathes = append(si.pathes, newPath)
// 				nxtBlock.Alive()
// 			}
// 		}
//
// 		var directSuccesor *cfg.Block
// 		if directSuccesorPC > si.cfg.Blocks[len(si.cfg.Blocks)-1].LastStmt().PC() {
// 			directSuccesor = nil
// 		} else {
// 			directSuccesor = si.cfg.BlockMap(directSuccesorPC)
// 			newPath := append(curPath, directSuccesor.Stmt[0].PC())
// 			if !isRelunctant(newPath) {
// 				directSuccesor.Alive()
// 				si.pathes = append(si.pathes, newPath)
// 			}
// 		}
//
// 	}
// 	fmt.Println("End of resolveCFG")
// 	fmt.Printf("%s", si.cfg.String())
// 	os.Exit(0)
// }

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

		params := make([]*operation, 0)

		if op.IsDup() {
			err := st.dupN(op.DupNum())
			if err != nil {
				return nil, false, err
			}
			params = append(params, st.peek())
		} else if op.IsSwap() {
			err := st.dupN(op.SwapNum())
			if err != nil {
				return nil, false, err
			}
			params = append(params, st.peek())
		} else {
			params = st.popN(opera.paramSize)
		}

		if len(params) != opera.paramSize {
			return nil, false, fmt.Errorf("param size not match, op: %s, paramSize: %d, len(params): %d", op.String(), opera.paramSize, len(params))
		}

		opera.params = params

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

			log.Warn(fmt.Sprintf("resolve success at %s\n", opera.expand(0)))
			return dest, false, nil
		}

		if op == vm.JUMP || op == vm.JUMPI || op.IsDup() || op.IsSwap() {
			continue
		}

		if opera.pushbackSize > 0 {
			st.pushN(opera, opera.pushbackSize)
		}
	}

	return nil, false, fmt.Errorf("unhandle error")
}
