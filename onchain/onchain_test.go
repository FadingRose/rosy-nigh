package onchain

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestOnChainCallCode(t *testing.T) {
	db := NewOnChainDataBase()

	tcs := []struct {
		address  common.Address
		hash     common.Hash
		expected []byte
	}{
		{
			address:  common.HexToAddress("0xf75e354c5edc8efed9b59ee9f67a80845ade7d0c"),
			hash:     common.HexToHash(""),
			expected: []byte("0x3660008037602060003660003473273930d21e01ee25e4c219b63259d214872220a261235a5a03f21560015760206000f3"),
		},
	}

	for _, tc := range tcs {
		actual, err := db.ContractCode(tc.address, tc.hash)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(actual, tc.expected) {
			t.Errorf("expected %v, got %v", tc.expected, actual)
		}
	}
}

func TestOnChainCallCodeBuffer(t *testing.T) {
	tc := struct {
		address  common.Address
		hash     common.Hash
		expected []byte
	}{
		address:  common.HexToAddress("0xf75e354c5edc8efed9b59ee9f67a80845ade7d0c"),
		hash:     common.HexToHash(""),
		expected: []byte("0x3660008037602060003660003473273930d21e01ee25e4c219b63259d214872220a261235a5a03f21560015760206000f3"),
	}

	db := NewOnChainDataBase()
	data, err := db.ContractCode(tc.address, tc.hash)
	// here we get bytecode from etherscan, hash it
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	hasher := crypto.NewKeccakState()
	hasher.Write(data)
	buf := common.Hash{}
	hasher.Read(buf[:])

	tc.hash = buf

	// concurrency access to the same db, all should get the same result from cached data
	// if cache is not working, it will cause API limit from etherscan
	for i := 0; i < 100; i++ {
		go func() {
			actual, err := db.ContractCode(tc.address, tc.hash)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !bytes.Equal(actual, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, actual)
			}
		}()
	}
}
