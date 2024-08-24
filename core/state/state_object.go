package state

import (
	"bytes"
	"fadingrose/rosy-nigh/core/types"
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

type Storage map[common.Hash]common.Hash

func (s Storage) Copy() Storage {
	return maps.Clone(s)
}

type stateObject struct {
	db       *StateDB
	address  common.Address
	addrHash common.Hash         // hash of ethereum address of the account
	origin   *types.StateAccount // Account original data without any change applied, nil means it was not existent
	data     types.StateAccount

	code []byte // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage entries that have been accessed within the current block
	dirtyStorage   Storage // Storage entries that have been modified within the current transaction
	pendingStorage Storage // Storage entries that have been modified within the current block

	// uncommittedStorage tracks a set of storage entries that have been modified
	// but not yet committed since the "last commit operation", along with their
	// original values before mutation.
	//
	// Specifically, the commit will be performed after each transaction before
	// the byzantium fork, therefore the map is already reset at the transaction
	// boundary; however post the byzantium fork, the commit will only be performed
	// at the end of block, this set essentially tracks all the modifications
	// made within the block.
	uncommittedStorage Storage

	// This is an EIP-6780 flag indicating whether the object is eligible for
	// self-destruct according to EIP-6780. The flag could be set either when
	// the contract is just created within the current transaction, or when the
	// object was previously existent and is being deployed as a contract within
	// the current transaction.
	newContract bool
}

func newObject(db *StateDB, addr common.Address, acct *types.StateAccount) *stateObject {
	origin := acct
	if acct == nil {
		acct = types.NewEmptyStateAccount()
	}

	return &stateObject{
		db:                 db,
		address:            addr,
		addrHash:           crypto.Keccak256Hash(addr[:]),
		origin:             origin,
		data:               *acct,
		originStorage:      make(Storage),
		dirtyStorage:       make(Storage),
		pendingStorage:     make(Storage),
		uncommittedStorage: make(Storage),
	}
}

// Code returns the contract code associated with this object, if any.
func (s *stateObject) Code() []byte {
	if len(s.code) != 0 {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), types.EmptyCodeHash.Bytes()) {
		return nil
	}

	code, err := s.db.online.ContractCode(s.address, common.BytesToHash(s.data.CodeHash))
	if err != nil {
		s.db.setError(fmt.Errorf("can't fetch code online hash %x for %s", s.data.CodeHash, s.address))
	}
	return code
}

// CodeSize returns the size of the contract code associated with this object,
// or zero if none. This method is an almost mirror of Code, but uses a cache
// inside the database to avoid loading codes seen recently.
func (s *stateObject) CodeSize() int {
	if len(s.code) != 0 {
		return len(s.code)
	}
	if bytes.Equal(s.CodeHash(), types.EmptyCodeHash.Bytes()) {
		return 0
	}
	size, err := s.db.online.ContractCodeSize(s.address, common.BytesToHash(s.data.CodeHash))
	if err != nil {
		s.db.setError(fmt.Errorf("can't fetch code size online hash %x for %s", s.data.CodeHash, s.address))
	}
	return size
}

// GetState retrieves a value associated with the given storage key.
func (s *stateObject) GetState(key common.Hash) common.Hash {
	value, _ := s.getState(key)
	return value
}

// getState retrieves a value associated with the given storage key, along with
// its original value.
func (s *stateObject) getState(key common.Hash) (common.Hash, common.Hash) {
	origin := s.GetCommittedState(key)
	value, dirty := s.dirtyStorage[key]
	if dirty {
		return value, origin
	}
	return origin, origin
}

// GetCommittedState retrieves the value associated with the specific key
// without any mutations caused in the current execution.
func (s *stateObject) GetCommittedState(key common.Hash) common.Hash {
	// If we have a pending write or clean cached, return that
	if value, pending := s.pendingStorage[key]; pending {
		return value
	}
	if value, cached := s.originStorage[key]; cached {
		return value
	}
	// If the object was destructed in *this* block (and potentially resurrected),
	// the storage has been cleared out, and we should *not* consult the previous
	// database about any storage values. The only possible alternatives are:
	//   1) resurrect happened, and new slot values were set -- those should
	//      have been handles via pendingStorage above.
	//   2) we don't have new values, and can deliver empty response back
	if _, destructed := s.db.stateObjectsDestruct[s.address]; destructed {
		s.originStorage[key] = common.Hash{} // track the empty slot as origin value
		return common.Hash{}
	}
	// If no live objects are available, attempt to use snapshots
	var (
		value common.Hash
	)
	s.originStorage[key] = value
	return value
}

// Getters
func (s *stateObject) Address() common.Address {
	return s.address
}

func (s *stateObject) Nonce() uint64 {
	return s.data.Nonce
}

func (s *stateObject) Balance() *uint256.Int {
	return s.data.Balance
}

func (s *stateObject) CodeHash() []byte {
	return s.data.CodeHash
}

func (s *stateObject) Root() common.Hash {
	return s.data.Root
}
