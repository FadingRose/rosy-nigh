package asm

import "fadingrose/rosy-nigh/src/core/vm"

func DisAssembler(bs []byte) []vm.Instruction {
	inss := make([]vm.Instruction, 0)
	pc := uint64(0)
	for pc < uint64(len(bs)) {
		var ins vm.Instruction

		b := bs[pc]
		op := vm.OpCode(b)
		ins.OpCode = op
		ins.PC = pc

		if op.IsPush() {
			offset := int(op - vm.PUSH1 + 1)
			if pc+uint64(offset)+1 >= uint64(len(bs)) {

				// append zero value to avoid panic
				trails := make([]byte, pc+uint64(offset)+1-uint64(len(bs)))
				bs = append(bs, trails...)
			}
			data := bs[pc+1 : pc+uint64(offset)+1]
			ins.Operand = data
			pc += uint64(offset) + 1
		} else {
			pc++
		}

		inss = append(inss, ins)
	}

	return inss
}

func Assembler(inss []vm.Instruction) []byte {
	bs := make([]byte, 0)
	for _, ins := range inss {
		bs = append(bs, byte(ins.OpCode))
		bs = append(bs, ins.Operand...)
	}
	return bs
}
