package fuzz

import (
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/core"
	"fadingrose/rosy-nigh/core/state"
	"fadingrose/rosy-nigh/core/vm"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type Contract struct {
	ABI        abi.ABI
	StaticBin  []byte // static bytecode without depoy arguments
	DeployeBin []byte // static bytecode with deploy arguments
	RuntimeBin []byte // runtime bytecode
	Version    string
	Name       string
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

// Execute is the main entry point for the fuzz
func Execute(contractFolder string) error {
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
				res = append(res, contract)
			}
		}
		return res
	}()

	fmt.Println("Fuzzing Tasks: ", orphans)

	for _, contract := range targets {
		err := execute(contract)
		if err != nil {
			return fmt.Errorf("failed to execute fuzzing on contract %s: %w", contract.Name, err)
		}
	}

	return nil
}

// NOTE: Entry point for a fuzzing task
func execute(contract *Contract) error {
	fmt.Println("Fuzzing contract: ", contract.Name)

	statedb, blockCtx, chainConfig, config := CreateEVMRuntimeEnironment()
	host := NewFuzzHost(contract, statedb, blockCtx, chainConfig, config)
	host.RunForDeploy()
	return nil
}
