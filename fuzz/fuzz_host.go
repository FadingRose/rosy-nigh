package fuzz

import (
	"fadingrose/rosy-nigh/abi"
	"fadingrose/rosy-nigh/core"
	"fadingrose/rosy-nigh/core/state"
	"fadingrose/rosy-nigh/core/vm"
	"fadingrose/rosy-nigh/mutator"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type Mutator interface {
	GenerateArgs(abi.Method) ([]interface{}, []abi.Argument, []mutator.Seed)
}

type FuzzHost struct {
	StateDB      *state.StateDB
	BlockContext vm.BlockContext
	ChainConfig  params.ChainConfig
	EVMConfig    vm.Config

	SenderAddress common.Address

	Mutator

	Target *Contract

	Err error
}

func NewFuzzHost(target *Contract, statedb *state.StateDB, blockCtx vm.BlockContext, chainConfig params.ChainConfig, config vm.Config) *FuzzHost {
	sender := common.HexToAddress("0x1111111111111111111111111111111111111111")
	mutator := mutator.NewMutator(target.ABI)
	return &FuzzHost{
		StateDB:      statedb,
		BlockContext: blockCtx,
		ChainConfig:  chainConfig,
		EVMConfig:    config,

		SenderAddress: sender,

		Mutator: mutator,

		Target: target,
	}
}

func (host *FuzzHost) RunForDeploy() {
	nonce := host.StateDB.GetNonce(host.SenderAddress)
	amount := big.NewInt(0)
	gasLimit := uint64(100000000000)
	gasPrice := big.NewInt(0)
	signer := types.MakeSigner(&host.ChainConfig, big.NewInt(0), 0)
	basefee := big.NewInt(0)
	gaspool := core.GasPool(gasLimit)

	for {
		// 1. Generate parameters
		// 2. Pack it into a transaction and sign it to get a Message
		// 3. Create a new EVM and run the message
		args, _, _ := host.Mutator.GenerateArgs(host.Target.ABI.Constructor)
		codes, err := PackTxConstructor(host.Target.ABI, host.Target.StaticBin, args...)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		tx := types.NewContractCreation(nonce, amount, gasLimit, gasPrice, codes)
		msg, _ := core.TransactionToMessage(tx, signer, basefee)
		msg.From = host.SenderAddress
		msg.SkipAccountChecks = true // do NOT check nonce
		context := core.NewEVMTxContext(msg)

		var (
			rules    = host.ChainConfig.Rules(host.BlockContext.BlockNumber, host.BlockContext.Random != nil, host.BlockContext.Time)
			coinbase = host.BlockContext.Coinbase
		)

		host.StateDB.Prepare(rules, host.SenderAddress, coinbase, nil, vm.ActivePrecompiles(rules), nil)
		evm := vm.NewEVM(host.BlockContext, context, host.StateDB, &host.ChainConfig, host.EVMConfig)
		result, err := core.ApplyMessage(evm, msg, &gaspool)
		if err != nil {
			fmt.Println("Error: ", err)
			host.Err = err
			break
		}
		if result.Failed() {
			fmt.Println("Failed: ", result.Err)
			if result.Err == vm.ErrExecutionReverted {
				revertData := result.Revert()
				if len(revertData) > 0 {
					fmt.Println("Revert with data: ", revertData)
				}
			}
			host.Err = result.Err
		}

		if host.Err == nil {
			break
		}
	}
}

func PackTxConstructor(abi abi.ABI, code []byte, args ...interface{}) ([]byte, error) {
	// Pack the constructor data
	// constructor's name should be empty, in old versions, it should as same as contract name
	data, err := abi.Pack("", args...)
	if err != nil {
		return nil, err
		// panic(fmt.Sprintf("failed to pack data: %s", err))
	}
	code = append(code, data...)
	return code, nil
}
