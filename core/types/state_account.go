package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// StateAccount is the Ethereum consensus representation of accounts.
// These objects are stored in the main account trie.
// See [Evm Runtime Environment](../../docs/archi.md)
type StateAccount struct {
	Nonce    uint64
	Balance  *uint256.Int
	Root     common.Hash // merkle root of the storage trie
	CodeHash []byte
}

// NewEmptyStateAccount constructs an empty state account.
func NewEmptyStateAccount() *StateAccount {
	return &StateAccount{
		Balance:  new(uint256.Int),
		Root:     EmptyRootHash,
		CodeHash: EmptyCodeHash.Bytes(),
	}
}
