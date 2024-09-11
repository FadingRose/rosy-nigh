package cfg

import (
	"fmt"
	"strings"
)

type JUMP_TYPE int

const (
	JUMP JUMP_TYPE = iota
	JUMPI_TRUE
	JUMPI_FALSE
)

type Checkpoint struct {
	PC_From uint64
	From    *Block
	JUMP_TYPE
	PC_To uint64
	To    *Block
}

type Path struct {
	Start_PC     uint64
	Start        *Block
	Checkpoints  []Checkpoint
	Terminate_PC uint64
	Terminate    *Block
}

func (p *Path) AddCheckpoint(pc_from uint64, from *Block, reason JUMP_TYPE, pc_to uint64, to *Block) {
	p.Checkpoints = append(p.Checkpoints, Checkpoint{pc_from, from, reason, pc_to, to})
}

func (p *Path) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("> 0x%x", p.Start_PC))
	for i, checkpoint := range p.Checkpoints {
		//	if i == 0 {
		//		builder.WriteString(fmt.Sprintf("> %x", checkpoint.PC_From))
		//	}

		arrow := ""
		switch checkpoint.JUMP_TYPE {
		case JUMP:
			arrow = ", "
		case JUMPI_FALSE:
			arrow = " -F> "
		case JUMPI_TRUE:
			arrow = " -T> "
		}

		builder.WriteString(fmt.Sprintf("%s0x%x", arrow, checkpoint.PC_To))

		if i == len(p.Checkpoints)-1 {
			builder.WriteString(" |")
		}
	}
	builder.WriteString("\n")

	return builder.String()
}
