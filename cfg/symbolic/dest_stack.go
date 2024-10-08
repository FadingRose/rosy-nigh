package symbolic

import "fmt"

type validDestFunc func(dest uint64) bool

type destinationStack struct {
	validDest validDestFunc
	data      []uint64
}

func NewDestinationStack(valid validDestFunc) *destinationStack {
	return &destinationStack{
		validDest: valid,
		data:      make([]uint64, 0),
	}
}

func (s *destinationStack) SafePush(dest uint64) {
	if s.validDest(dest) {
		s.data = append(s.data, dest)
	}
}

func (s *destinationStack) Pop() (uint64, bool) {
	if len(s.data) == 0 {
		return 0, false
	}
	dest := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return dest, true
}

func (s *destinationStack) Peek() (uint64, bool) {
	if len(s.data) == 0 {
		return 0, false
	}
	return s.data[len(s.data)-1], true
}

func (s *destinationStack) String() string {
	return fmt.Sprintf(">%v", s.data)
}
