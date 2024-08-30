package vm

// Memory implements a simple memory model for the ethereum virtual machine.
type Memory struct {
	store       []byte
	lastGasCost uint64
}

// NewMemory returns a new memory model.
func newMemory() *Memory {
	return &Memory{}
}

// Data returns the backing slice
func (m *Memory) Data() []byte {
	return m.store
}

// Len returns the length of the backing slice
func (m *Memory) Len() int {
	return len(m.store)
}

// Resize resizes the memory to size
func (m *Memory) Resize(size uint64) {
	if uint64(m.Len()) < size {
		m.store = append(m.store, make([]byte, size-uint64(m.Len()))...)
	}
}
