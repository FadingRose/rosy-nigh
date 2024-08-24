package state

import "github.com/ethereum/go-ethereum/common"

type Database interface {
	// ContractCode retrieves a particular contract's code.
	ContractCode(addr common.Address, codeHash common.Hash) ([]byte, error)

	// ContractCodeSize retrieves a particular contracts code's size.
	ContractCodeSize(addr common.Address, codeHash common.Hash) (int, error)
}
