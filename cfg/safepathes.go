package cfg

import (
	"encoding/binary"
	"fadingrose/rosy-nigh/log"
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

	maxdepth int
}

func (sp *safepathes) append(path []uint64) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if len(path) > sp.maxdepth {
		return
	}

	nn := path[len(path)-1]
	for i := 0; i < len(path)-1; i++ {
		if path[i] == nn {
			log.Warn(fmt.Sprintf("loop detected at %d of %v\n", nn, path))
			return
		}
	}

	hash := sp.hashPath(path)
	if _, exists := sp.pathHashes[hash]; !exists {
		sp.pathes = append(sp.pathes, path)
		sp.pathHashes[hash] = struct{}{}
	}
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
	var ret string
	for _, path := range sp.pathes {
		ret += fmt.Sprintf("%v\n", path)
	}
	return ret
}
