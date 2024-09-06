package onchain

import (
	"fadingrose/rosy-nigh/log"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
)

func loadCodeCache() map[common.Address][]byte {
	cacheDir := filepath.Join(".", ".cache", "statedb")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		os.MkdirAll(cacheDir, os.ModePerm)
		return make(map[common.Address][]byte)
	}

	files, err := os.ReadDir(cacheDir)
	if err != nil {
		log.Warn("read cachedb failed")
		return make(map[common.Address][]byte)
	}

	codecache := make(map[common.Address][]byte)

	// load code cache from .cache/statedb/<address>/code.bin
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		codebin := filepath.Join(cacheDir, file.Name(), "code.bin")
		code, err := os.ReadFile(codebin)
		if err != nil {
			log.Warn(fmt.Sprintf("read code.bin failed at %s", codebin))
			continue
		}
		address := common.HexToAddress(file.Name())
		codecache[address] = code
	}

	return codecache
}

func appendCodeCache(address common.Address, code []byte) {
	go func() {
		if hasCached(address) {
			return
		}
		cache(address, code)
	}()
}

func hasCached(address common.Address) bool {
	cacheDir := filepath.Join(".", ".cache", "statedb", address.Hex())
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return false
	}
	return true
}

func cache(address common.Address, code []byte) {
	cacheDir := filepath.Join(".", ".cache", "statedb", address.Hex())
	os.MkdirAll(cacheDir, os.ModePerm)
	cacheFile := filepath.Join(cacheDir, "code.bin")
	os.WriteFile(cacheFile, code, os.ModePerm)
}
