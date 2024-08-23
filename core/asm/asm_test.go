package asm

import (
	"encoding/hex"
	"testing"
)

// Tests disassembling instructions
func TestInstructionIterator(t *testing.T) {
	for i, tc := range []struct {
		want    int
		code    string
		wantErr string
	}{
		{2, "61000000", ""},                             // valid code
		{0, "6100", "incomplete push instruction at 0"}, // invalid code
		{2, "5900", ""},                                 // push0
		{0, "", ""},                                     // empty

	} {
		var (
			have    int
			code, _ = hex.DecodeString(tc.code)
			it      = NewInstructionIterator(code)
		)
		for it.Next() {
			have++
		}
		haveErr := ""
		if it.Error() != nil {
			haveErr = it.Error().Error()
		}
		if haveErr != tc.wantErr {
			t.Errorf("test %d: encountered error: %q want %q", i, haveErr, tc.wantErr)
			continue
		}
		if have != tc.want {
			t.Errorf("wrong instruction count, have %d want %d", have, tc.want)
		}
	}
}
