package fuzz

import (
	"encoding/hex"
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
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
	creationPath := filepath.Join(fileDir, name+".bin-creation")
	creatorPath := filepath.Join(fileDir, name+".address-creator")

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

	creationContent, err := loadFile(creationPath)
	if err != nil {
		return nil, err
	}

	creatorContent, err := os.ReadFile(creatorPath)
	if err != nil {
		return nil, err
	}
	creator := common.BytesToAddress(creatorContent)
	return &Contract{
		ABI:         abiContent,
		StaticBin:   binContent,
		CreationBin: creationContent,
		RuntimeBin:  nil,
		Name:        name,
		Creator:     creator,
	}, nil
}

func loadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Warn("file not exists", "path", path)
		return nil, nil
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

func saveDeployedBin(contractAddress string, bin []byte, abi string, creator string) (string, error) {
	cacheDir := filepath.Join(".", ".cache", "creation", contractAddress)
	// make sure ./.cache/creation/<contractAddress>/ folder exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		os.MkdirAll(cacheDir, os.ModePerm)
	} else {
		return "", err
	}

	// write to ./.cache/creation/<contractAddress>/<contractAddress>.bin-creation
	cacheCreation := filepath.Join(cacheDir, contractAddress+".bin-creation")
	err := os.WriteFile(cacheCreation, bin, os.ModePerm)
	if err != nil {
		return "", err
	}

	// write to ./.cache/creation/<contractAddress>/<contractAddress>.abi
	cacheABI := filepath.Join(cacheDir, contractAddress+".abi")
	err = os.WriteFile(cacheABI, []byte(abi), os.ModePerm)
	if err != nil {
		return "", err
	}

	cacheCreator := filepath.Join(cacheDir, contractAddress+".address-creator")
	err = os.WriteFile(cacheCreator, []byte(creator), os.ModePerm)
	if err != nil {
		return "", err
	}
	return cacheDir, nil
}

func hasCached(contractAddress string) (string, bool) {
	cacheDir := filepath.Join(".", ".cache", "creation", contractAddress)
	cacheCration := filepath.Join(cacheDir, contractAddress+".bin-creation")
	cacheABI := filepath.Join(cacheDir, contractAddress+".abi")
	cacheCreator := filepath.Join(cacheDir, contractAddress+".address-creator")

	_, err := os.Stat(cacheCration)
	if err != nil {
		return cacheDir, false
	}
	_, err = os.Stat(cacheABI)
	if err != nil {
		return cacheDir, false
	}
	_, err = os.Stat(cacheCreator)
	if err != nil {
		return cacheDir, false
	}

	return cacheDir, true
}
