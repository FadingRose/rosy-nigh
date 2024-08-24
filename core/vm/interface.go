package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StateDB interface {
	// creations
	CreateAccount(addr common.Address)
	CreateContract(addr common.Address, code []byte)

	// world state for an Address
	GetNonce(addr common.Address) uint64
	SetNonce(addr common.Address, nonce uint64)

	SubBalance(addr common.Address, amount *uint256.Int)
	AddBalance(addr common.Address, amount *uint256.Int)
	GetBalance(addr common.Address) *uint256.Int
	SetBalance(addr common.Address, amount *uint256.Int)

	GetCodeHash(addr common.Address) common.Hash
	GetCode(addr common.Address) []byte
	SetCode(addr common.Address, code []byte)
	GetCodeSize(addr common.Address) uint

	// support for SSTORE, SLOAD
	GetCommittedState(addr common.Address, key common.Hash) common.Hash
	GetState(addr common.Address, key common.Hash) common.Hash
	SetState(addr common.Address, key common.Hash, value common.Hash)
	GetStorageRoot(addr common.Address) common.Hash

	// Gas
	// AddRefund(uint64)
	// SubRefund(uint64)
	// GetRefund() uint64
}
