package cfg

import (
	"encoding/hex"
	"os"
	"testing"
)

func TestCFG(t *testing.T) {
	data, err := loadFile("../testdata/0x0addedfee0e8a65c9a60067b9fe0f24af96da51d_reentrancy/0.6.1/PEG.bin")
	if err != nil {
		t.Fatal(err)
	}
	cfg := NewCFG(data)
	t.Logf("%s", cfg.String())
}

func loadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = parseStrToBytes(string(data))
	return data, nil
}

func parseStrToBytes(s string) []byte {
	// two char -> one byte
	var b []byte
	data, err := hex.DecodeString(s)
	if err == nil {
		b = data
	}
	return b
}
