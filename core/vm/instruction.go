package vm

type Instruction struct {
	PC uint64
	OpCode
	Operand []byte
}
