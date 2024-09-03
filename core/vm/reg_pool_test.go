package vm

import "testing"

func TestRegPoolRebuildDup(t *testing.T) {
	rp := NewRegPool()
	inss := []struct {
		pc           uint64
		depth        uint64
		op           OpCode
		paramSize    int
		pushbackSize int
	}{
		{
			pc:           0,
			depth:        1,
			op:           PUSH1,
			paramSize:    0,
			pushbackSize: 1,
		},
		{
			pc:           1,
			depth:        1,
			op:           PUSH2,
			paramSize:    0,
			pushbackSize: 1,
		},
		{
			pc:           2,
			depth:        1,
			op:           DUP2,
			paramSize:    1,
			pushbackSize: 1,
		},
		{
			pc:           2,
			depth:        1,
			op:           DUP2,
			paramSize:    1,
			pushbackSize: 1,
		},
	}

	for _, ins := range inss {
		rp.Append(ins.pc, ins.depth, ins.op, ins.paramSize, ins.pushbackSize)
	}

	rp.rebuild()

	t.Logf("Rebuild:\n%s", rp.String())
}

func TestRegPoolRebuildSwap(t *testing.T) {
	rp := NewRegPool()
	inss := []struct {
		pc           uint64
		depth        uint64
		op           OpCode
		paramSize    int
		pushbackSize int
	}{
		{
			pc:           0,
			depth:        1,
			op:           PUSH1,
			paramSize:    0,
			pushbackSize: 1,
		},
		{
			pc:           1,
			depth:        1,
			op:           PUSH2,
			paramSize:    0,
			pushbackSize: 1,
		},
		{
			pc:           2,
			depth:        1,
			op:           SWAP1,
			paramSize:    2,
			pushbackSize: 1,
		},
		{
			pc:           2,
			depth:        1,
			op:           SWAP1,
			paramSize:    2,
			pushbackSize: 1,
		},
	}

	for _, ins := range inss {
		rp.Append(ins.pc, ins.depth, ins.op, ins.paramSize, ins.pushbackSize)
	}

	rp.rebuild()

	t.Logf("Rebuild:\n%s", rp.String())
}
