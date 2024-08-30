package vm

import (
	"fadingrose/rosy-nigh/core/tracing"
	"fadingrose/rosy-nigh/core/types"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type (
	// CanTransferFunc is the signature of a transfer guard function
	CanTransferFunc func(StateDB, common.Address, *uint256.Int) bool
	// TransferFunc is the signature of a transfer function
	TransferFunc func(StateDB, common.Address, common.Address, *uint256.Int)
	// GetHashFunc returns the n'th block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(uint64) common.Hash
)

// BlockContext provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type BlockContext struct {
	// CanTransfer returns whether the account contains
	// sufficient ether to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers ether from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Block information
	Coinbase    common.Address // Provides information for COINBASE
	GasLimit    uint64         // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        uint64         // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
	BaseFee     *big.Int       // Provides information for BASEFEE (0 if vm runs with NoBaseFee flag and 0 gas price)
	BlobBaseFee *big.Int       // Provides information for BLOBBASEFEE (0 if vm runs with NoBaseFee flag and 0 blob gas price)
	Random      *common.Hash   // Provides information for PREVRANDAO
}

// Config are the configuration options for the Interpreter
type Config struct {
	Tracer                  *tracing.Hooks
	NoBaseFee               bool  // Forces the EIP-1559 baseFee to 0 (needed for 0 price calls)
	EnablePreimageRecording bool  // Enables recording of SHA3/keccak preimages
	ExtraEips               []int // Additional EIPS that are to be enabled
}

type EVM struct {
	StateDB
	// Context provides auxiliary blockchain related information
	depth   int
	Context BlockContext
	TxContext
	// chainConfig contains information about the current chain
	chainConfig *params.ChainConfig
	// virtual machine configuration options used to initialise the
	// evm.
	Config Config

	// global (to this context) ethereum virtual machine
	// used throughout the execution of the tx.
	interpreter EVMInterpreter
	// chain rules contains the chain rules for the current epoch
	chainRules params.Rules

	// last time the evm interprereter runs with ScopeContext
	ScopeContext *ScopeContext
}

// TxContext provides the EVM with information about a transaction.
// All fields can change between transactions.
type TxContext struct {
	// Message information
	Origin       common.Address      // Provides information for ORIGIN
	GasPrice     *big.Int            // Provides information for GASPRICE (and is used to zero the basefee if NoBaseFee is set)
	BlobHashes   []common.Hash       // Provides information for BLOBHASH
	BlobFeeCap   *big.Int            // Is used to zero the blobbasefee if NoBaseFee is set
	AccessEvents *state.AccessEvents // Capture all state accesses for this tx
}

// ChainConfig returns the environment's chain configuration
func (evm *EVM) ChainConfig() *params.ChainConfig { return evm.chainConfig }

type codeAndHash struct {
	code []byte
	hash common.Hash
}

// Create creates a new contract using code as deployment code.
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *uint256.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), evm.StateDB.GetNonce(caller.Address()))
	return evm.create(caller, &codeAndHash{code: code}, gas, value, contractAddr, CREATE)
}

// create creates a new contract using code as deployment code.
func (evm *EVM) create(caller ContractRef, codeAndHash *codeAndHash, gas uint64, value *uint256.Int, address common.Address, typ OpCode) (ret []byte, createAddress common.Address, leftOverGas uint64, err error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit. Now Max Call Create Depth is 1024
	if evm.depth > int(params.CallCreateDepth) {
		return nil, common.Address{}, gas, ErrDepth
	}

	if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}

	nonce := evm.StateDB.GetNonce(caller.Address())
	if nonce+1 < nonce {
		return nil, common.Address{}, gas, ErrNonceUintOverflow
	}
	evm.StateDB.SetNonce(caller.Address(), nonce+1)
	// // We add this to the access list _before_ taking a snapshot. Even if the
	// // creation fails, the access-list change should not be rolled back.
	// if evm.chainRules.IsEIP2929 {
	// 	evm.StateDB.AddAddressToAccessList(address)
	// }

	// Ensure there's no existing contract already at the designated address.
	// Account is regarded as existent if any of these three conditions is met:
	// - the nonce is non-zero
	// - the code is non-empty
	// - the storage is non-empty
	contractHash := evm.StateDB.GetCodeHash(address)
	storageRoot := evm.StateDB.GetStorageRoot(address)
	if evm.StateDB.GetNonce(address) != 0 ||
		(contractHash != (common.Hash{}) && contractHash != types.EmptyCodeHash) || // non-empty code
		(storageRoot != (common.Hash{}) && storageRoot != types.EmptyRootHash) { // non-empty storage
		if evm.Config.Tracer != nil && evm.Config.Tracer.OnGasChange != nil {
			evm.Config.Tracer.OnGasChange(gas, 0, tracing.GasChangeCallFailedExecution)
		}
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}

	// Create a new account on the state only if the object was not present.
	// It might be possible the contract code is deployed to a pre-existent
	// account with non-zero balance.
	// WARNING: Snapshot not supported now, revert always return 0
	snapshot := evm.StateDB.Snapshot()
	if !evm.StateDB.Exist(address) {
		evm.StateDB.CreateAccount(address)
	}

	// CreateContract means that regardless of whether the account previously existed
	// in the state trie or not, it _now_ becomes created as a _contract_ account.
	// This is performed _prior_ to executing the initcode,  since the initcode
	// acts inside that account.
	evm.StateDB.CreateContract(address)

	// EIP: Simplifies state management, by removing empty accounts, reducing state size and enhancing protocol efficiency.
	if evm.chainRules.IsEIP158 {
		evm.StateDB.SetNonce(address, 1)
	}
	evm.Context.Transfer(evm.StateDB, caller.Address(), address, value)

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, AccountRef(address), value, gas)
	contract.SetCodeOptionalHash(&address, codeAndHash)
	contract.IsDeployment = true

	// Charge the contract creation init gas in verkle mode
	if evm.chainRules.IsEIP4762 {
		if !contract.UseGas(evm.AccessEvents.ContractCreateInitGas(address, value.Sign() != 0), evm.Config.Tracer, tracing.GasChangeWitnessContractInit) {
			err = ErrOutOfGas
		}
	}

	if err == nil {
		ret, err = evm.interpreter.Run(contract, nil, false)
	}

	// Check whether the max code size has been exceeded, assign err if the case.
	if err == nil && evm.chainRules.IsEIP158 && len(ret) > params.MaxCodeSize {
		err = ErrMaxCodeSizeExceeded
	}

	// Reject code starting with 0xEF if EIP-3541 is enabled.
	if err == nil && len(ret) >= 1 && ret[0] == 0xEF && evm.chainRules.IsLondon {
		err = ErrInvalidCode
	}

	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil {
		if !evm.chainRules.IsEIP4762 {
			createDataGas := uint64(len(ret)) * params.CreateDataGas
			if !contract.UseGas(createDataGas, evm.Config.Tracer, tracing.GasChangeCallCodeStorage) {
				err = ErrCodeStoreOutOfGas
			}
		} else {
			// Contract creation completed, touch the missing fields in the contract
			if !contract.UseGas(evm.AccessEvents.AddAccount(address, true), evm.Config.Tracer, tracing.GasChangeWitnessContractCreation) {
				err = ErrCodeStoreOutOfGas
			}

			if err == nil && len(ret) > 0 && !contract.UseGas(evm.AccessEvents.CodeChunksRangeGas(address, 0, uint64(len(ret)), uint64(len(ret)), true), evm.Config.Tracer, tracing.GasChangeWitnessCodeChunk) {
				err = ErrCodeStoreOutOfGas
			}
		}

		if err == nil {
			evm.StateDB.SetCode(address, ret)
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally,
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil && (evm.chainRules.IsHomestead || err != ErrCodeStoreOutOfGas) {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != ErrExecutionReverted {
			contract.UseGas(contract.Gas, evm.Config.Tracer, tracing.GasChangeCallFailedExecution)
		}
	}

	return ret, address, contract.Gas, err
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *uint256.Int) (ret []byte, leftOverGas uint64, err error) {
	return nil, 0, nil
}

// NOTE: Expose to Interpreter interface

// IncreaseCallStackDepth expose for Interpreter interface
func (evm *EVM) IncreaseCallStackDepth() {
	evm.depth++
}

// DecreaseCallStackDepth expose for Interpreter interface
func (evm *EVM) DecreaseCallStackDepth() {
	evm.depth--
}

func (evm *EVM) GetDepth() int {
	return evm.depth
}

func (evm *EVM) GetChainRules() params.Rules {
	return evm.chainRules
}
