package terminal

import (
	"bytes"
	"fadingrose/rosy-nigh/core/vm"
	"testing"
)

type MockClient struct{}

func (m *MockClient) RegExpand(pc uint64) (string, error) {
	return "mock reg expand", nil
}

func (m *MockClient) RegOpcode(op vm.OpCode) (string, error) {
	return "mock opcode", nil
}

type StringReader struct {
	buf *bytes.Buffer
}

func newStringReader() *StringReader {
	return &StringReader{buf: new(bytes.Buffer)}
}

func (sr *StringReader) WriteString(s string) (int, error) {
	return sr.buf.WriteString(s)
}

func (sr *StringReader) Read(p []byte) (int, error) {
	return sr.buf.Read(p)
}

func TestRegCmds(t *testing.T) {
	mock := &MockClient{}
	stdin := newStringReader()
	_, err := stdin.WriteString(".reg 0x1\n")
	if err != nil {
		t.Fatal(err)
	}
	term := NewTerminal(mock, stdin)
	term.Run()
}
