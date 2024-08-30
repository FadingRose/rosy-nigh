package vm

import (
	"sync"

	"github.com/holiman/uint256"
)

type Stack struct {
	data []*Reg
}

var stackPool = sync.Pool{
	New: func() interface{} {
		return &Stack{data: make([]*Reg, 0, 16)}
	},
}

func newstack() *Stack {
	return &Stack{}
}

func returnStack(s *Stack) {
	s.data = s.data[:0]
	stackPool.Put(s)
}

// Data returns the underlying uint256.Int array.
func (st *Stack) Data() []uint256.Int {
	var data []uint256.Int
	for _, reg := range st.data {
		data = append(data, reg.Data)
	}
	return data
}

func (st *Stack) len() int {
	return len(st.data)
}

func (st *Stack) checkParam(offset int) *Reg {
	if offset >= len(st.data) {
		return nil
	}
	return st.data[len(st.data)-offset-1]
}

func (st *Stack) push(r *Reg) {
	// NOTE push limit (1024) is checked in baseCheck
	st.data = append(st.data, r)
}

func (st *Stack) pop() (ret uint256.Int) {
	ret = st.data[len(st.data)-1].Data
	st.data = st.data[:len(st.data)-1]
	return
}

func (st *Stack) peek() *uint256.Int {
	return &st.data[st.len()-1].Data
}

// Back returns the n'th item in stack
func (st *Stack) Back(n int) *uint256.Int {
	return &st.data[st.len()-n-1].Data
}
