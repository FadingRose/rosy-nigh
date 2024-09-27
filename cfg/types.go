package cfg

import (
	"fmt"
	"strings"

	"github.com/holiman/uint256"
)

// For a JUMPI, whether it's FALSE or TRUE branch is coverd?
type JMPIBranch map[int32][2]bool

type JMPBranch map[int32]bool

type CondBranch = int

const (
	FalseBranch CondBranch = iota
	TrueBranch
)

func (jumpi JMPIBranch) String() string {
	var ret strings.Builder
	for index, branch := range jumpi {
		var (
			falseBranch = " "
			trueBranch  = " "
		)
		if branch[FalseBranch] {
			falseBranch = "x"
		}
		if branch[TrueBranch] {
			trueBranch = "x"
		}
		ret.WriteString(fmt.Sprintf("dest: 0x%x: false-branch:[%s] true-branch:[%s]\n", index, falseBranch, trueBranch))
	}
	return ret.String()
}

func (jump JMPBranch) String() string {
	var ret strings.Builder
	for index, covered := range jump {
		branch := " "
		if covered {
			branch = "x"
		}
		ret.WriteString(fmt.Sprintf("dest: 0x%x branch:[%s]\n", index, branch))
	}
	return ret.String()
}

type SlotCoverage struct {
	SSTORECover int
	SSTORETotal int
	SLOADCover  int
	SLOADTotal  int
}

type AccessType int

const (
	Unknown AccessType = iota
	Read
	Write
)

func (at AccessType) string() string {
	switch at {
	case Read:
		return "R"
	case Write:
		return "W"
	default:
		return "Unknown"
	}
}

type SlotAccess struct {
	AccessType
	Key   uint256.Int
	Value uint256.Int
}

func (s SlotAccess) String() string {
	return fmt.Sprintf("[%s] %s -> %s", s.AccessType.string(), s.Key.Hex(), s.Value.Hex())
}
