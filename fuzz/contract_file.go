package fuzz

import (
	"encoding/hex"
	"fadingrose/rosy-nigh/abi"
	"os"
	"path/filepath"
	"strings"
)

func loadContractsFromDir(dir string) ([]*Contract, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, file := range files {
		if !file.IsDir() {
			name := strings.Split(file.Name(), ".")[0]
			if !contains(names, name) {
				names = append(names, name)
			}
		}
	}

	var contracts []*Contract
	for _, n := range names {
		contract, err := loadContractFromFile(dir, n)
		if err != nil {
			return nil, err
		}
		contracts = append(contracts, contract)
	}

	return contracts, nil
}

// fileDir is a directory, which contains
// .abi and .bin files
func loadContractFromFile(fileDir string, name string) (*Contract, error) {
	abiPath := filepath.Join(fileDir, name+".abi")
	binPath := filepath.Join(fileDir, name+".bin")

	abifile, err := os.Open(abiPath)
	if err != nil {
		return nil, err
	}
	abiContent, err := abi.JSON(abifile)
	if err != nil {
		return nil, err
	}

	binContent, err := loadFile(binPath)
	if err != nil {
		return nil, err
	}

	return &Contract{
		ABI:        abiContent,
		StaticBin:  binContent,
		DeployeBin: nil,
		RuntimeBin: nil,
		Name:       name,
	}, nil
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
