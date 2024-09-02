package vm

type RegKey struct {
	index [3]uint64 // depth -> pc -> loop
	reg   *Reg
}

type RegPool struct {
	regkeyList      []RegKey
	loopLookUpTable map[[2]uint64]uint64 // [depth,pc] -> loop
}

func NewRegPool() *RegPool {
	return &RegPool{
		regkeyList:      make([]RegKey, 0),
		loopLookUpTable: make(map[[2]uint64]uint64, 1024),
	}
}

// Append appends a new register to the register pool.
func (rp *RegPool) Append(pc uint64, depth uint64, op OpCode, paramSize int) *Reg {
	loop := rp.lookup(pc, depth)

	index := [3]uint64{depth, pc, loop}
	reg := newReg(index, op, paramSize)
	rp.regkeyList = append(rp.regkeyList, RegKey{
		index: index,
		reg:   reg,
	})
	return reg
}

func (rp *RegPool) lookup(pc uint64, depth uint64) uint64 {
	query := [2]uint64{depth, pc}
	if loop, ok := rp.loopLookUpTable[query]; ok {
		rp.loopLookUpTable[query] = loop + 1
	} else {
		rp.loopLookUpTable[query] = 0
	}
	return rp.loopLookUpTable[query]
}

// TODO: Implement Rebuild, it will rebuild the regkeylist to a Tree structure.
// TEST: RegPool Verification
func (rp *RegPool) Rebuild() {
}
