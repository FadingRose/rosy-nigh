package tracing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// OpContext provides the context at which the opcode is being
// executed in, including the memory, stack and various contract-level information.
type OpContext interface {
	MemoryData() []byte
	StackData() []uint256.Int
	Caller() common.Address
	Address() common.Address
	CallValue() *uint256.Int
	CallInput() []byte
}

type (
	// OpcodeHook is invoked just prior to the execution of an opcode.
	OpcodeHook = func(pc uint64, op byte, gas, cost uint64, scope OpContext, rData []byte, depth int, err error)
	// FaultHook is invoked when an error occurs during the execution of an opcode.
	FaultHook = func(pc uint64, op byte, gas, cost uint64, scope OpContext, depth int, err error)
	// EnterHook is invoked when the processing of a message starts.
	//
	// Take note that EnterHook, when in the context of a live tracer, can be invoked
	// outside of the `OnTxStart` and `OnTxEnd` hooks when dealing with system calls,
	// see [OnSystemCallStartHook] and [OnSystemCallEndHook] for more information.
	EnterHook = func(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	// ExitHook is invoked when the processing of a message ends.
	// `revert` is true when there was an error during the execution.
	// Exceptionally, before the homestead hardfork a contract creation that
	// ran out of gas when attempting to persist the code to database did not
	// count as a call failure and did not cause a revert of the call. This will
	// be indicated by `reverted == false` and `err == ErrCodeStoreOutOfGas`.
	//
	// Take note that ExitHook, when in the context of a live tracer, can be invoked
	// outside of the `OnTxStart` and `OnTxEnd` hooks when dealing with system calls,
	// see [OnSystemCallStartHook] and [OnSystemCallEndHook] for more information.
	ExitHook = func(depth int, output []byte, gasUsed uint64, err error, reverted bool)
)

// GasChangeHook is invoked when the gas changes.
type GasChangeHook = func(old, new uint64, reason GasChangeReason)

type Hooks struct {
	OnGasChange GasChangeHook
	OnOpcode    OpcodeHook
	OnFault     FaultHook

	OnEnter EnterHook
	OnExit  ExitHook
}

// BalanceChangeReason is used to indicate the reason for a balance change, useful
// for tracing and reporting.
type BalanceChangeReason byte

const (
	BalanceChangeUnspecified BalanceChangeReason = 0

	// Issuance
	// BalanceIncreaseRewardMineUncle is a reward for mining an uncle block.
	BalanceIncreaseRewardMineUncle BalanceChangeReason = 1
	// BalanceIncreaseRewardMineBlock is a reward for mining a block.
	BalanceIncreaseRewardMineBlock BalanceChangeReason = 2
	// BalanceIncreaseWithdrawal is ether withdrawn from the beacon chain.
	BalanceIncreaseWithdrawal BalanceChangeReason = 3
	// BalanceIncreaseGenesisBalance is ether allocated at the genesis block.
	BalanceIncreaseGenesisBalance BalanceChangeReason = 4

	// Transaction fees
	// BalanceIncreaseRewardTransactionFee is the transaction tip increasing block builder's balance.
	BalanceIncreaseRewardTransactionFee BalanceChangeReason = 5
	// BalanceDecreaseGasBuy is spent to purchase gas for execution a transaction.
	// Part of this gas will be burnt as per EIP-1559 rules.
	BalanceDecreaseGasBuy BalanceChangeReason = 6
	// BalanceIncreaseGasReturn is ether returned for unused gas at the end of execution.
	BalanceIncreaseGasReturn BalanceChangeReason = 7

	// DAO fork
	// BalanceIncreaseDaoContract is ether sent to the DAO refund contract.
	BalanceIncreaseDaoContract BalanceChangeReason = 8
	// BalanceDecreaseDaoAccount is ether taken from a DAO account to be moved to the refund contract.
	BalanceDecreaseDaoAccount BalanceChangeReason = 9

	// BalanceChangeTransfer is ether transferred via a call.
	// it is a decrease for the sender and an increase for the recipient.
	BalanceChangeTransfer BalanceChangeReason = 10
	// BalanceChangeTouchAccount is a transfer of zero value. It is only there to
	// touch-create an account.
	BalanceChangeTouchAccount BalanceChangeReason = 11

	// BalanceIncreaseSelfdestruct is added to the recipient as indicated by a selfdestructing account.
	BalanceIncreaseSelfdestruct BalanceChangeReason = 12
	// BalanceDecreaseSelfdestruct is deducted from a contract due to self-destruct.
	BalanceDecreaseSelfdestruct BalanceChangeReason = 13
	// BalanceDecreaseSelfdestructBurn is ether that is sent to an already self-destructed
	// account within the same tx (captured at end of tx).
	// Note it doesn't account for a self-destruct which appoints itself as recipient.
	BalanceDecreaseSelfdestructBurn BalanceChangeReason = 14
)

// GasChangeReason is used to indicate the reason for a gas change, useful
// for tracing and reporting.
//
// There is essentially two types of gas changes, those that can be emitted once per transaction
// and those that can be emitted on a call basis, so possibly multiple times per transaction.
//
// They can be recognized easily by their name, those that start with `GasChangeTx` are emitted
// once per transaction, while those that start with `GasChangeCall` are emitted on a call basis.
type GasChangeReason byte

const (
	GasChangeUnspecified GasChangeReason = 0

	// GasChangeTxInitialBalance is the initial balance for the call which will be equal to the gasLimit of the call. There is only
	// one such gas change per transaction.
	GasChangeTxInitialBalance GasChangeReason = 1
	// GasChangeTxIntrinsicGas is the amount of gas that will be charged for the intrinsic cost of the transaction, there is
	// always exactly one of those per transaction.
	GasChangeTxIntrinsicGas GasChangeReason = 2
	// GasChangeTxRefunds is the sum of all refunds which happened during the tx execution (e.g. storage slot being cleared)
	// this generates an increase in gas. There is at most one of such gas change per transaction.
	GasChangeTxRefunds GasChangeReason = 3
	// GasChangeTxLeftOverReturned is the amount of gas left over at the end of transaction's execution that will be returned
	// to the chain. This change will always be a negative change as we "drain" left over gas towards 0. If there was no gas
	// left at the end of execution, no such even will be emitted. The returned gas's value in Wei is returned to caller.
	// There is at most one of such gas change per transaction.
	GasChangeTxLeftOverReturned GasChangeReason = 4

	// GasChangeCallInitialBalance is the initial balance for the call which will be equal to the gasLimit of the call. There is only
	// one such gas change per call.
	GasChangeCallInitialBalance GasChangeReason = 5
	// GasChangeCallLeftOverReturned is the amount of gas left over that will be returned to the caller, this change will always
	// be a negative change as we "drain" left over gas towards 0. If there was no gas left at the end of execution, no such even
	// will be emitted.
	GasChangeCallLeftOverReturned GasChangeReason = 6
	// GasChangeCallLeftOverRefunded is the amount of gas that will be refunded to the call after the child call execution it
	// executed completed. This value is always positive as we are giving gas back to the you, the left over gas of the child.
	// If there was no gas left to be refunded, no such even will be emitted.
	GasChangeCallLeftOverRefunded GasChangeReason = 7
	// GasChangeCallContractCreation is the amount of gas that will be burned for a CREATE.
	GasChangeCallContractCreation GasChangeReason = 8
	// GasChangeContractCreation is the amount of gas that will be burned for a CREATE2.
	GasChangeCallContractCreation2 GasChangeReason = 9
	// GasChangeCallCodeStorage is the amount of gas that will be charged for code storage.
	GasChangeCallCodeStorage GasChangeReason = 10
	// GasChangeCallOpCode is the amount of gas that will be charged for an opcode executed by the EVM, exact opcode that was
	// performed can be check by `OnOpcode` handling.
	GasChangeCallOpCode GasChangeReason = 11
	// GasChangeCallPrecompiledContract is the amount of gas that will be charged for a precompiled contract execution.
	GasChangeCallPrecompiledContract GasChangeReason = 12
	// GasChangeCallStorageColdAccess is the amount of gas that will be charged for a cold storage access as controlled by EIP2929 rules.
	GasChangeCallStorageColdAccess GasChangeReason = 13
	// GasChangeCallFailedExecution is the burning of the remaining gas when the execution failed without a revert.
	GasChangeCallFailedExecution GasChangeReason = 14
	// GasChangeWitnessContractInit is the amount charged for adding to the witness during the contract creation initialization step
	GasChangeWitnessContractInit GasChangeReason = 15
	// GasChangeWitnessContractCreation is the amount charged for adding to the witness during the contract creation finalization step
	GasChangeWitnessContractCreation GasChangeReason = 16
	// GasChangeWitnessCodeChunk is the amount charged for touching one or more contract code chunks
	GasChangeWitnessCodeChunk GasChangeReason = 17

	// GasChangeIgnored is a special value that can be used to indicate that the gas change should be ignored as
	// it will be "manually" tracked by a direct emit of the gas change event.
	GasChangeIgnored GasChangeReason = 0xFF
)
