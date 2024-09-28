package cfg

import (
	"fmt"
	"sync"
	"testing"
)

func TestReadWrite(t *testing.T) {
	pathes := [][]uint64{
		{1, 2, 3},
	}
	sp := &safepathes{
		pathes:     pathes,
		mu:         sync.Mutex{},
		pathHashes: map[uint64]struct{}{},
	}
	cur, ok := sp.get()
	if !ok {
		t.Fatal("cur should be ok")
	}
	if len(cur) != 3 {
		t.Fatal("cur lens should be 3")
	}
	if true {
		np := append(cur, 4)
		sp.append(np)
	}

	if true {
		np := append(cur, 5)
		sp.append(np)
	}
	// np1 := append(cur, 4)
	// sp.append(np1)
	// np2 := append(cur, 5)
	// sp.append(np2)

	cnt := uint64(6)
	for i := 0; i < 100; i++ {

		np := append(cur, cnt)
		sp.append(np)
		cnt++
	}

	res1, _ := sp.get()
	res2, _ := sp.get()
	fmt.Printf("res1: %v\nres2: %v\n", res1, res2)
}
