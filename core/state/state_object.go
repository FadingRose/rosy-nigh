package state

import (
	"bytes"
	"fadingrose/rosy-nigh/core/tracing"
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	// Flag whether the account was marked as self-destructed. The self-destructed
	// account is still accessible in the scope of same transaction.
	selfDestructed bool
	// Cache flags.
	dirtyCode bool // true if the code was updated
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

// AddBalance adds amount to s's balance.
// It is used to add funds to the destination account of a transfer.
func (s *stateObject) AddBalance(amount *uint256.Int, reason tracing.BalanceChangeReason) {
	// EIP161: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.IsZero() {
		if s.empty() {
			s.touch()
		}
		return
	}
	s.SetBalance(new(uint256.Int).Add(s.Balance(), amount), reason)
}

// SubBalance removes amount from s's balance.
// It is used to remove funds from the origin account of a transfer.
func (s *stateObject) SubBalance(amount *uint256.Int, reason tracing.BalanceChangeReason) {
	if amount.IsZero() {
		return
	}
	s.SetBalance(new(uint256.Int).Sub(s.Balance(), amount), reason)
}

func (s *stateObject) SetBalance(amount *uint256.Int, reason tracing.BalanceChangeReason) {
	s.db.journal.append(balanceChange{
		account: &s.address,
		prev:    new(uint256.Int).Set(s.data.Balance),
	})
	// TODO:Use tracer replaces this
	// if s.db.logger != nil && s.db.logger.OnBalanceChange != nil {
	// 	s.db.logger.OnBalanceChange(s.address, s.Balance().ToBig(), amount.ToBig(), reason)
	// }
	s.setBalance(amount)
}

func (s *stateObject) setBalance(amount *uint256.Int) {
	s.data.Balance = amount
}

// empty returns whether the account is considered empty.
func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.IsZero() && bytes.Equal(s.data.CodeHash, types.EmptyCodeHash.Bytes())
}

func (s *stateObject) touch() {
	s.db.journal.append(touchChange{
		account: &s.address,
	})
	if s.address == ripemd {
		// Explicitly put it in the dirty-cache, which is otherwise generated from
		// flattened journals.
		s.db.journal.dirty(s.address)
	}
}

func (s *stateObject) markSelfdestructed() {
	s.selfDestructed = true
}

func (s *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := s.Code()
	s.db.journal.append(codeChange{
		account:  &s.address,
		prevhash: s.CodeHash(),
		prevcode: prevcode,
	})
	// use tracer replaces this
	// if s.db.logger != nil && s.db.logger.OnCodeChange != nil {
	// 	s.db.logger.OnCodeChange(s.address, common.BytesToHash(s.CodeHash()), prevcode, codeHash, code)
	// }
	s.setCode(codeHash, code)
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	s.data.CodeHash = codeHash[:]
	s.dirtyCode = true
}

func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.append(nonceChange{
		account: &s.address,
		prev:    s.data.Nonce,
	})
	// use tracer replaces this
	// if s.db.logger != nil && s.db.logger.OnNonceChange != nil {
	// 	s.db.logger.OnNonceChange(s.address, s.data.Nonce, nonce)
	// }
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.data.Nonce = nonce
}

// SetState updates a value in account storage.
func (s *stateObject) SetState(key, value common.Hash) {
	// If the new value is the same as old, don't set. Otherwise, track only the
	// dirty changes, supporting reverting all of it back to no change.
	prev, origin := s.getState(key)
	if prev == value {
		return
	}
	// New value is different, update and journal the change
	s.db.journal.append(storageChange{
		account:   &s.address,
		key:       key,
		prevvalue: prev,
		origvalue: origin,
	})
	// if s.db.logger != nil && s.db.logger.OnStorageChange != nil {
	// 	s.db.logger.OnStorageChange(s.address, key, prev, value)
	// }
	s.setState(key, value, origin)
}

// setState updates a value in account dirty storage. The dirtiness will be
// removed if the value being set equals to the original value.
func (s *stateObject) setState(key common.Hash, value common.Hash, origin common.Hash) {
	// Storage slot is set back to its original value, undo the dirty marker
	if value == origin {
		delete(s.dirtyStorage, key)
		return
	}
	s.dirtyStorage[key] = value
}
