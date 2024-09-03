package vm

import (
	"bufio"
	"fadingrose/rosy-nigh/log"
	"fmt"
	"os"
	"strings"
)

type RegKey struct {
	index [3]uint64 // depth -> pc -> loop
	reg   *Reg
}

func (rk *RegKey) Instance() *Reg {
	return rk.reg
}

// Duplicate supports opcode DUP
func (rk *RegKey) Duplicate() *RegKey {
	return &RegKey{
		index: rk.index,
		reg:   rk.reg.Duplicate(),
	}
}

func (rk *RegKey) String() string {
	return rk.reg.String()
}

type RegPool struct {
	regkeyList      []RegKey
	loopLookUpTable map[[2]uint64]uint64 // [depth,pc] -> loop
}

func (rp *RegPool) String() string {
	var builder strings.Builder

	for _, rk := range rp.regkeyList {
		builder.WriteString(rk.String())
	}

	return builder.String()
}

func NewRegPool() *RegPool {
	return &RegPool{
		regkeyList:      make([]RegKey, 0),
		loopLookUpTable: make(map[[2]uint64]uint64, 1024),
	}
}

// Append appends a new register to the register pool.
func (rp *RegPool) Append(pc uint64, depth uint64, op OpCode, paramSize int, pushbackSize int) *Reg {
	loop := rp.lookup(pc, depth)

	index := [3]uint64{depth, pc, loop}
	reg := newReg(index, op, paramSize, pushbackSize)
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
func (rp *RegPool) rebuild() {
	log.Debug("Rebuilding the register pool")
	log.Debug(rp.String())
	st := newSymbolicStack()
	for _, rk := range rp.regkeyList {
		// 1. read params, popN from stack
		//     1.a params = 0, skip 1.
		//     1.b params > 0, popN(params), then setup the reg's L, M ...
		// 2. read pushback
		if rk.reg.op.IsPush() {
			st.push(rk)
			log.Debug(st.String())
			continue
		}

		if rk.reg.op == POP {
			st.pop()
			continue
		}

		if opWithoutPushBack(rk.reg.op) {
			continue
		}

		// 1.b there are two special cases
		// 1.b.1 DUP
		var params []RegKey

		if rk.reg.op.IsDup() {
			st.DupN(rk.reg.op.DupNum())
			params = append(params, st.Peek())
		} else if rk.reg.op.IsSwap() {
			st.SwapN(rk.reg.op.SwapNum())
			params = append(params, st.Peek())
		} else {
			params = st.PopN(rk.reg.paramSize)
		}

		if params == nil {
			log.Warn("Rebuild StackUnderFlow", "rk", rk.String())
		}

		rk.Instance().setupParams(params)
		st.push(rk)

		log.Debug(st.String())
	}
}

// TODO: impl debug
func (rp *RegPool) Debug() {
	rp.rebuild()
	fmt.Println(rp.String())
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		input = input[:len(input)-1]
		if input == ".exit" || input == ".q" {
			break
		}
	}
}

func opWithoutPushBack(op OpCode) bool {
	if op.IsLog() || op == CALLDATACOPY || op == CODECOPY || op == EXTCODECOPY || op == RETURNDATACOPY ||
		op == POP || op == MSTORE || op == MSTORE8 || op == JUMP || op == JUMPI || op == JUMPDEST ||
		op == TSTORE || op == MCOPY || op == RETURN || op == REVERT || op == SELFDESTRUCT {
		return true
	}
	return false
}

type symbolicStack struct {
	data []RegKey
}

func newSymbolicStack() *symbolicStack {
	return &symbolicStack{data: make([]RegKey, 0, 16)}
}

func (st *symbolicStack) push(r RegKey) {
	st.data = append(st.data, r)
}

func (st *symbolicStack) pop() RegKey {
	if len(st.data) == 0 {
		return RegKey{}
	}
	r := st.data[len(st.data)-1]
	st.data = st.data[:len(st.data)-1]
	return r
}

// nPop make sure the return value is in the order of the stack
// [L,R -> pop(2) -> [L,R]
func (st *symbolicStack) PopN(n int) []RegKey {
	if len(st.data) < n {
		log.Warn("Rebuild StackUnderFlow", "want to pop", n, "but only", len(st.data))
		return nil // StackUnderFlow
	}
	ret := make([]RegKey, n)
	for i := n - 1; i >= 0; i-- {
		ret[i] = st.pop()
	}
	return ret
}

// Nil means reach the end of the stack
func (st *symbolicStack) Peek() RegKey {
	if len(st.data) == 0 {
		return RegKey{} // Reach the end of the stack
	}
	return st.data[len(st.data)-1]
}

// DupN duplicates the nth element from stack
func (s *symbolicStack) DupN(n int) {
	rk := s.data[len(s.data)-n]
	dup := rk.Duplicate()
	s.push(*dup)
}

func (s *symbolicStack) SwapN(n int) {
	n++
	s.data[len(s.data)-1], s.data[len(s.data)-n] = s.data[len(s.data)-n], s.data[len(s.data)-1]
}

func (s *symbolicStack) String() string {
	ret := "["
	for i := len(s.data) - 1; i >= 0; i-- {
		ret += s.data[i].reg.op.String() + " "
	}

	return ret
}
