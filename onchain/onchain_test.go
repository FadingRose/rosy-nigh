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
			expected: []byte("3660008037602060003660003473273930d21e01ee25e4c219b63259d214872220a261235a5a03f21560015760206000f3"),
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

func TestGetTx(t *testing.T) {
	txhash := "0x33f4594b6eeb9fdf2f3b8e2c7bfb1335bddf577dc6ec2154e3f558c36be8f4a4"
	api := ApiKeys()[ETH]
	eth := Chain(ETH)
	from, input, to, err := eth.GetTx(txhash, api)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	} else {
		t.Logf("from: %s, to: %s, input: %s", from, to, input)
	}
}

func TestGetCreation(t *testing.T) {
	contractAddress := "0x0addedfee0e8a65c9a60067b9fe0f24af96da51d"
	api := ApiKeys()[ETH]
	eth := Chain(ETH)
	creator, txhash, err := eth.GetCreation(contractAddress, api)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	} else {
		t.Logf("creator: %s, txhash: %s", creator, txhash)
	}
}

func TestGetABI(t *testing.T) {
	contractAddress := "0x0addedfee0e8a65c9a60067b9fe0f24af96da51d"
	api := ApiKeys()[ETH]
	eth := Chain(ETH)
	abi, err := eth.GetABI(contractAddress, api)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	} else {
		t.Logf("abi: %s", abi)
	}
}
