package terminal

import "testing"

func TestFlagUint64Parse(t *testing.T) {
	flag := FlagBase[uint64]{Name: "test"}
	name, val, _ := flag.Parse("0x1")
	if name != "test" {
		t.Errorf("Expected name to be 'test', got %s", name)
	}
	if val != uint64(1) {
		t.Errorf("Expected value to be 1, got %d", val)
	}
}
