package symbolic

import "fmt"

type stack struct {
	data []*operation
}

func newStack() *stack {
	return &stack{
		data: make([]*operation, 0),
	}
}

func (s *stack) peek() *operation {
	if len(s.data) == 0 {
		return nil
	}
	return s.data[len(s.data)-1]
}

func (s *stack) push(p *operation) {
	s.data = append(s.data, p)
}

func (s *stack) pushN(p *operation, n int) {
	for i := 0; i < n; i++ {
		s.push(p)
	}
}

func (s *stack) pop() *operation {
	if len(s.data) == 0 {
		return nil
	}
	p := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return p
}

// popN make sure the return value is in the order of the stack
// [L,R -> pop(2) -> [L,R]
func (s *stack) popN(n int) []*operation {
	if len(s.data) < n {
		return nil // StackUnderFlow
	}
	ret := make([]*operation, n)
	for i := n - 1; i >= 0; i-- {
		ret[i] = s.pop()
	}
	return ret
}

// DupN duplicates the nth element from stack
func (s *stack) dupN(n int) error {
	if len(s.data) < n {
		return fmt.Errorf("stack underflow")
	}
	p := s.data[len(s.data)-n]
	dup := p.dup()
	s.push(dup)
	return nil
}

// SwapN swaps the nth element from stack with the top element
func (s *stack) SwapN(n int) error {
	if len(s.data) < n {
		return fmt.Errorf("stack underflow")
	}

	n++
	s.data[len(s.data)-1], s.data[len(s.data)-n] = s.data[len(s.data)-n], s.data[len(s.data)-1]
	return nil
}

func (s *stack) string() string {
	ret := "["
	for _, p := range s.data {
		ret += p.op.String() + ","
	}
	ret += "\n"
	return ret
}
