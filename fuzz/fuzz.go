package fuzz

import (
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/core"
	"fadingrose/rosy-nigh/core/state"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/log"
	"fadingrose/rosy-nigh/onchain"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type Contract struct {
	ABI         abi.ABI
	StaticBin   []byte // static bytecode without depoy arguments
	CreationBin []byte // static bytecode with deploy arguments
	RuntimeBin  []byte // runtime bytecode
	Version     string
	Name        string
	Creator     common.Address
}

func CreateEVMRuntimeEnironment() (statedb *state.StateDB, blockCtx vm.BlockContext, chainConfig params.ChainConfig, config vm.Config) {
	statedb = state.NewStateDB()

	// BlockContext
	blockCtx = vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,

		GetHash: core.GetHashFn(nil, nil),

		Coinbase:    common.Address{},
		BlockNumber: big.NewInt(1),
		Time:        uint64(1), // TO ENABLE SHANGHAI FORK
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(0),
		BlobBaseFee: big.NewInt(0),
		GasLimit:    0,

		Random: &common.Hash{}, // TO ENABLE SHANGHAI FORK
	}
	// ChainConfig
	shanghaiTime := uint64(0)
	chainConfig = params.ChainConfig{
		ChainID:      big.NewInt(1),
		LondonBlock:  big.NewInt(0),
		ShanghaiTime: &shanghaiTime,
	}
	// Config
	config = vm.Config{
		Tracer:                  nil,
		NoBaseFee:               true,
		EnablePreimageRecording: false,
	}
	return statedb, blockCtx, chainConfig, config
}

// PrepareOnchainCache is used to prepare the onchain cache
func PrepareOnchainCache(onchainAddress string) (cacheFolder string, err error) {
	if cacheDir, ok := hasCached(onchainAddress); ok {
		log.Info(fmt.Sprintf("cache find at %s", cacheDir))
		return cacheDir, nil
	}

	// 1. Get CreationTx and ABI
	// 2. From Creationtx, get the deployed bytecode(means with the creation arguments)
	var (
		api     = onchain.ApiKeys()[onchain.ETH]
		eth     = onchain.Chain(onchain.ETH)
		txhash  string
		creator common.Address
		// from          common.Address
		creationCodes []byte
		// to            common.Address
		abi string
	)

	creator, txhash, err = eth.GetCreation(onchainAddress, api)
	if err != nil {
		return "", fmt.Errorf("failed to get contract creation: %w", err)
	}

	abi, err = eth.GetABI(onchainAddress, api)
	if err != nil {
		return "", fmt.Errorf("failed to get contract abi: %w", err)
	}

	_, creationCodes, _, err = eth.GetTx(txhash, api)
	if err != nil {
		return "", fmt.Errorf("failed to get contract transaction: %w", err)
	}

	return saveDeployedBin(onchainAddress, creationCodes, abi, creator.Hex())
}

// Execute is the main entry point for the local fuzz
func Execute(contractFolder string, debug bool) error {
	// Load and resolve contracts
	contracts, err := loadContractsFromDir(contractFolder)
	if err != nil {
		return fmt.Errorf("failed to load contracts: %w", err)
	}
	inherit := NewInheritancer(contracts)
	orphans := inherit.FindInheritance()

	targets := func() []*Contract {
		var res []*Contract
		for _, contract := range contracts {
			if contains(orphans, contract.Name) {
				if len(contract.StaticBin) > 0 || len(contract.CreationBin) > 0 {
					res = append(res, contract) // ignore interface
				}
			}
		}
		return res
	}()

	log.Info("Fuzzing Tasks: ", "orphans", orphans)

	for _, contract := range targets {
		err := execute(contract, debug)
		if err != nil {
			return fmt.Errorf("failed to execute fuzzing on contract %s: %w", contract.Name, err)
		}
	}

	return nil
}

// NOTE: Entry point for a fuzzing task
func execute(contract *Contract, debug bool) error {
	var host *FuzzHost
	defer func() {
		if debug {
			host.Debug()
		}
	}()

	statedb, blockCtx, chainConfig, config := CreateEVMRuntimeEnironment()
	host = NewFuzzHost(contract, statedb, blockCtx, chainConfig, config)

	log.Info("Fuzzing contract: ", "name", contract.Name)
	// host.RunForDeploy()
	host.RunForDeployOnchain()
	return nil
}
