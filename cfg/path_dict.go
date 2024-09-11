package cfg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type PathDict struct {
	dict map[string]bool
}

func NewPathDict() *PathDict {
	return &PathDict{
		dict: make(map[string]bool),
	}
}

// IsNewPathDiscovered returns TRUE if the path is a new path
func (pd *PathDict) IsNewPathDiscovered(path *Path) (flag bool) {
	pathes := distract(path)

	flag = false
	for _, p := range pathes {
		hash := hashPath(p)
		if _, ok := pd.dict[hash]; !ok {
			pd.dict[hash] = true
			flag = true
		}
	}

	return flag
}

// for example, path is 0x1 -T> 0x2 -F> 0x3
// we should split it to all the sub-pathes
// 0x1 -T> 0x2
// 0x1 -T> 0x2 -F> 0x3
func distract(path *Path) [][]uint64 {
	var ret [][]uint64

	// If path.Checkpoints is empty, try to load val from Start_PC and Terminate_PC
	if len(path.Checkpoints) == 0 {
		if path.Start_PC != path.Terminate_PC {
			panic("Path Checkpoints is empty, but Start_PC != Terminate_PC")
		}
		ret = append(ret, []uint64{path.Start_PC, uint64(JUMP)})
		return ret
	}

	// Add root fisrt
	var cur []uint64
	cur = append(cur, path.Checkpoints[0].PC_From)
	cur = append(cur, uint64(path.Checkpoints[0].JUMP_TYPE))

	for i := 0; i < len(path.Checkpoints); i++ {
		sub := cur
		ret = append(ret, sub)

		cur = append(cur, path.Checkpoints[i].PC_To)
		cur = append(cur, uint64(path.Checkpoints[i].JUMP_TYPE))
	}

	return ret
}

func hashPath(p []uint64) string {
	pathStr := fmt.Sprintf("%v", p)
	hash := sha256.Sum256([]byte(pathStr))
	return hex.EncodeToString(hash[:])
}
