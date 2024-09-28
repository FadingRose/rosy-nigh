package symbolic

import (
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"strings"

	"github.com/holiman/uint256"
)

type Operation struct {
	o *operation
}

func NewOperation(paramSize int, pushbackSize int, op vm.OpCode, pc uint64, val *uint256.Int, exec func(me *operation) *uint256.Int) *Operation {
	return &Operation{
		o: &operation{
			paramSize:    paramSize,
			pushbackSize: pushbackSize,
			op:           op,
			pc:           pc,
			val:          val,
			exec:         exec,
		},
	}
}

func (O *Operation) instance() *operation {
	return O.o
}

type operation struct {
	paramSize    int
	pushbackSize int
	op           vm.OpCode
	pc           uint64
	val          *uint256.Int
	params       []*operation

	cp *operation

	exec func(me *operation) *uint256.Int
}

func (o *operation) string() string {
	val := "<nil>"
	if o.val != nil {
		val = o.val.Hex()
	}
	return fmt.Sprintf("op: %s, paramSize: %d, pushbackSize: %d, val: %s\n", o.op.String(), o.paramSize, o.pushbackSize, val)
}

func (o *operation) dup() *operation {
	return &operation{
		paramSize:    o.paramSize,
		pushbackSize: o.pushbackSize,
		op:           o.op,
		pc:           o.pc,
		val:          o.val,
		cp:           o.cp,
	}
}

func (o *operation) solve() *uint256.Int {
	if len(o.params) != o.paramSize {
		return nil
	}

	for _, p := range o.params {
		if p.val == nil {
			// fmt.Printf("unsolve at %s\n", o.expand(0))
			return nil
		}
	}

	return o.exec(o)
}

func (o *operation) expand(depth int) string {
	if depth > 1024 {
		log.Warn("reg expand overflow, depth > 1024")
		return ""
	}

	to := o
	for to.cp != nil {
		to = to.cp
	}

	var builder strings.Builder
	// indentation for the tree structure
	indent := strings.Repeat(".", 6)
	indents := strings.Repeat(indent, depth)

	nameWithOp := func(o *operation) string {
		return fmt.Sprintf("%s %s", o.name(), o.op.String())
	}

	// isSkip returns true if we do NOT expand the next node
	isSkip := func(op vm.OpCode) bool {
		control := []vm.OpCode{vm.STOP, vm.POP, vm.JUMPDEST}
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

	val := "<nil>"
	if to.val != nil {
		val = to.val.Hex()
	}

	if depth == 0 {
		builder.WriteString(fmt.Sprintf("%s\n", nameWithOp(to)))
	} else {
		// Write the current node
		builder.WriteString(fmt.Sprintf("%s   └── %s <- %s\n", indents, nameWithOp(to), val))
	}

	if isSkip(to.op) {
		return builder.String()
	}

	for _, it := range to.params {
		builder.WriteString(it.expand(depth + 1))
	}

	return builder.String()
}

func (o *operation) name() string {
	return fmt.Sprintf("[%d]", o.pc)
}
