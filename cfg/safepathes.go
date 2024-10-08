package cfg

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

type safepathes struct {
	mu         sync.Mutex
	pathes     [][]uint64
	pathHashes map[uint64]struct{}
	index      int32

	twice map[uint64]struct{}

	maxdepth int
}

func (sp *safepathes) append(path []uint64) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if len(path) > sp.maxdepth {
		return fmt.Errorf("path too long: %d", len(path))
	}

	hash := sp.hashPath(path)

	// nn := path[len(path)-1]
	// for i := 0; i < len(path)-1; i++ {
	// 	if path[i] == nn {
	// 		// HACK: if we do NOT detect loop, give a second change to entry a loop
	// 		// pathes: 352
	// 		if _, exists := sp.twice[nn]; !exists {
	// 			sp.twice[nn] = struct{}{}
	// 			sp.pathes = append(sp.pathes, path)
	// 			// return fmt.Errorf(fmt.Sprintf("loop detected at %d of %v\n", nn, path))
	// 			return nil
	// 		} else {
	// 			return fmt.Errorf(fmt.Sprintf("loop detected at %d of %v\n", nn, path))
	// 		}
	//
	// 		// // HACK: detect loop
	// 		// log.Warn(fmt.Sprintf("loop detected at %d of %v\n", nn, path))
	// 		// return nil
	// 	}
	// }

	if _, exists := sp.pathHashes[hash]; !exists {
		sp.pathes = append(sp.pathes, path)
		sp.pathHashes[hash] = struct{}{}
	}

	return nil
}

func (sp *safepathes) len() int {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return len(sp.pathes)
}

func (sp *safepathes) get() ([]uint64, bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	// fmt.Println("sp.index", sp.index)
	idx := int(atomic.LoadInt32(&sp.index))

	if idx >= len(sp.pathes) {
		return nil, false
	}

	defer atomic.AddInt32(&sp.index, 1)

	ret := make([]uint64, len(sp.pathes[idx]))
	copy(ret, sp.pathes[idx])
	return ret, true
}

func (sp *safepathes) hashPath(path []uint64) uint64 {
	h := fnv.New64a()
	for _, pc := range path {
		binary.Write(h, binary.LittleEndian, pc)
	}
	return h.Sum64()
}

func (sp *safepathes) string() string {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	maxlen := 0
	for _, path := range sp.pathes {
		if len(path) > maxlen {
			maxlen = len(path)
		}
	}
	return fmt.Sprintf("pathes discovered: %d, max path len: %d\n", len(sp.pathes), maxlen)
}

func (sp *safepathes) status() (idx, total int) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	idx = int(atomic.LoadInt32(&sp.index))
	total = len(sp.pathes)
	return idx, total
}
