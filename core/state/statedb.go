package state

import (
	"fadingrose/rosy-nigh/core/tracing"
	"fadingrose/rosy-nigh/onchain"
	"fmt"
	"slices"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type revision struct {
	id           int
	journalIndex int
}

type StateDB struct {
	// online Database
	online Database

	// state objects
	stateObjects map[common.Address]*stateObject

	// Transient storage
	transientStorage transientStorage

	// Preimages occurred seen by VM in the scope of block.
	preimages map[common.Hash][]byte

	// Per-transaction access list
	accessList *accessList

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int

	// The tx context and all occurred logs in the scope of transaction.
	thash   common.Hash
	txIndex int
	logs    map[common.Hash][]*types.Log
	logSize uint
	// This map holds 'deleted' objects. An object with the same address
	// might also occur in the 'stateObjects' map due to account
	// resurrection. The account value is tracked as the original value
	// before the transition. This map is populated at the transaction
	// boundaries.
	stateObjectsDestruct map[common.Address]*stateObject

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be
	// returned by StateDB.Commit. Notably, this error is also shared
	// by all cached state objects in case the database failure occurs
	// when accessing state of accounts.
	dbErr error
	// The refund counter, also used by state transitioning.
	refund uint64
}

func NewStateDB() *StateDB {
	onlinedb := onchain.NewOnChainDataBase()
	return &StateDB{
		online:               onlinedb,
		stateObjects:         make(map[common.Address]*stateObject),
		stateObjectsDestruct: make(map[common.Address]*stateObject),
		accessList:           newAccessList(),
		journal:              newJournal(),
		preimages:            make(map[common.Hash][]byte),
		logs:                 make(map[common.Hash][]*types.Log),
	}
}

// CreateAccount explicitly creates a new state object, assuming that the
// account did not previously exist in the state. If the account already
// exists, this function will silently overwrite it which might lead to a
// consensus bug eventually.
func (s *StateDB) CreateAccount(addr common.Address) {
	s.createObject(addr)
}

// CreateContract is used whenever a contract is created. This may be preceded
// by CreateAccount, but that is not required if it already existed in the
// state due to funds sent beforehand.
// This operation sets the 'newContract'-flag, which is required in order to
// correctly handle EIP-6780 'delete-in-same-transaction' logic.
func (s *StateDB) CreateContract(addr common.Address) {
	obj := s.getStateObject(addr)
	if !obj.newContract {
		obj.newContract = true
		s.journal.append(createContractChange{account: addr})
	}
}

// AddBalance adds amount to the account associated with addr.
func (s *StateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount, reason)
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (s *StateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount, reason)
	}
}

// GetNonce retrieves the nonce from the given address or 0 if object not found
func (s *StateDB) GetNonce(addr common.Address) uint64 {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0
}

// GetBalance retrieves the balance from the given address or 0 if object not found
func (s *StateDB) GetBalance(addr common.Address) *uint256.Int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return common.U2560
}

func (s *StateDB) GetCode(addr common.Address) []byte {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code()
	}
	return nil
}

func (s *StateDB) GetCodeSize(addr common.Address) int {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.CodeSize()
	}
	return 0
}

func (s *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return common.BytesToHash(stateObject.CodeHash())
	}
	return common.Hash{}
}

// GetState retrieves the value associated with the specific key.
// This do NOT returns the persistent state
func (s *StateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(hash)
	}
	return common.Hash{}
}

// GetCommittedState retrieves the value associated with the specific key
// without any mutations caused in the current execution.
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetCommittedState(hash)
	}
	return common.Hash{}
}

// GetStorageRoot retrieves the storage root from the given address or empty
// if object not found.
func (s *StateDB) GetStorageRoot(addr common.Address) common.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Root()
	}
	return common.Hash{}
}

func (s *StateDB) setStateObject(object *stateObject) {
	s.stateObjects[object.Address()] = object
}

// createObject creates a new state object. The assumption is held there is no
// existing account with the given address, otherwise it will be silently overwritten.
func (s *StateDB) createObject(addr common.Address) *stateObject {
	obj := newObject(s, addr, nil)
	s.journal.append(createObjectChange{account: &addr})
	s.setStateObject(obj)
	return obj
}

// getOrNewStateObject retrieves a state object or create a new state object if nil.
func (s *StateDB) getOrNewStateObject(addr common.Address) *stateObject {
	obj := s.getStateObject(addr)
	if obj == nil {
		obj = s.createObject(addr)
	}
	return obj
}

// getStateObject retrieves a state object given by the address, returning nil if
// the object is not found or was deleted in this execution context.
func (s *StateDB) getStateObject(addr common.Address) *stateObject {
	// Prefer live objects if any is available
	if obj := s.stateObjects[addr]; obj != nil {
		// fmt.Println("state object is found")
		return obj
	}
	// Short circuit if the account is already destructed in this block.
	if _, ok := s.stateObjectsDestruct[addr]; ok {
		// fmt.Println("state object is already destructed")
		return nil
	}

	// TODO: online fuzzing suport

	// Create New Object and insert into the live set
	obj := newObject(s, addr, nil)
	s.setStateObject(obj)
	// fmt.Println("state object is created")
	return obj
}

// setError remembers the first non-nil error it is called with.
func (s *StateDB) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

// Exist reports whether the given account exists in state.
// Notably this should also return true for self-destructed accounts.
func (s *StateDB) Exist(addr common.Address) bool {
	return s.getStateObject(addr) != nil
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *StateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(s.validRevisions), func(i int) bool {
		return s.validRevisions[i].id >= revid
	})
	if idx == len(s.validRevisions) || s.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := s.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	s.journal.revert(s, snapshot)
	s.validRevisions = s.validRevisions[:idx]
	// log.Info("Reverting to snapshot", "revid", revid, "validRevisions", s.validRevisions)
}

// TODO: add a  fucntion, revert to the init state
func (s *StateDB) RevertToInitState(initState int) int {
	s.RevertToSnapshot(initState)
	return s.Snapshot()
}

// Snapshot returns an identifier for the current revision of the state.
func (s *StateDB) Snapshot() int {
	id := s.nextRevisionId
	s.nextRevisionId++
	s.validRevisions = append(s.validRevisions, revision{id, s.journal.length()})
	// log.Info("Taking Snapshot", "revision", s.nextRevisionId-1, "validRevisions", s.validRevisions)
	return id
}

// AddAddressToAccessList adds the given address to the access list
func (s *StateDB) AddAddressToAccessList(addr common.Address) {
	if s.accessList.AddAddress(addr) {
		s.journal.append(accessListAddAccountChange{&addr})
	}
}

func (s *StateDB) AddLog(log *types.Log) {
	s.journal.append(addLogChange{txhash: s.thash})

	log.TxHash = s.thash
	log.TxIndex = uint(s.txIndex)
	log.Index = s.logSize
	// TODO: use tracer replace this
	// if s.logger != nil && s.logger.OnLog != nil {
	// 	s.logger.OnLog(log)
	// }
	s.logs[s.thash] = append(s.logs[s.thash], log)
	s.logSize++
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *StateDB) AddPreimage(hash common.Hash, preimage []byte) {
	if _, ok := s.preimages[hash]; !ok {
		s.journal.append(addPreimageChange{hash: hash})
		s.preimages[hash] = slices.Clone(preimage)
	}
}

// AddRefund adds gas to the refund counter
func (s *StateDB) AddRefund(gas uint64) {
	s.journal.append(refundChange{prev: s.refund})
	s.refund += gas
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (s *StateDB) SubRefund(gas uint64) {
	s.journal.append(refundChange{prev: s.refund})
	if gas > s.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, s.refund))
	}
	s.refund -= gas
}

// AddressInAccessList returns true if the given address is in the access list.
func (s *StateDB) AddressInAccessList(addr common.Address) bool {
	return s.accessList.ContainsAddress(addr)
}

// AddSlotToAccessList adds the given (address, slot)-tuple to the access list
func (s *StateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	addrMod, slotMod := s.accessList.AddSlot(addr, slot)
	if addrMod {
		// In practice, this should not happen, since there is no way to enter the
		// scope of 'address' without having the 'address' become already added
		// to the access list (via call-variant, create, etc).
		// Better safe than sorry, though
		s.journal.append(accessListAddAccountChange{&addr})
	}
	if slotMod {
		s.journal.append(accessListAddSlotChange{
			address: &addr,
			slot:    &slot,
		})
	}
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (s *StateDB) Empty(addr common.Address) bool {
	so := s.getStateObject(addr)
	return so == nil || so.empty()
}

// GetRefund returns the current value of the refund counter.
func (s *StateDB) GetRefund() uint64 {
	return s.refund
}

// GetTransientState gets transient storage for a given account.
func (s *StateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return s.transientStorage.Get(addr, key)
}

// SelfDestruct marks the given account as selfdestructed.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after SelfDestruct.
func (s *StateDB) SelfDestruct(addr common.Address) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return
	}
	var (
		prev = new(uint256.Int).Set(stateObject.Balance())
		n    = new(uint256.Int)
	)
	s.journal.append(selfDestructChange{
		account:     &addr,
		prev:        stateObject.selfDestructed,
		prevbalance: prev,
	})
	// TODO: use tracer replace this
	// if s.logger != nil && s.logger.OnBalanceChange != nil && prev.Sign() > 0 {
	// 	s.logger.OnBalanceChange(addr, prev.ToBig(), n.ToBig(), tracing.BalanceDecreaseSelfdestruct)
	// }
	stateObject.markSelfdestructed()
	stateObject.data.Balance = n
}

func (s *StateDB) Selfdestruct6780(addr common.Address) {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return
	}
	if stateObject.newContract {
		s.SelfDestruct(addr)
	}
}

func (s *StateDB) HasSelfDestructed(addr common.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.selfDestructed
	}
	return false
}

// Prepare handles the preparatory steps for executing a state transition with.
// This method must be invoked before state transition.
//
// Berlin fork:
// - Add sender to access list (2929)
// - Add destination to access list (2929)
// - Add precompiles to access list (2929)
// - Add the contents of the optional tx access list (2930)
//
// Potential EIPs:
// - Reset access list (Berlin)
// - Add coinbase to access list (EIP-3651)
// - Reset transient storage (EIP-1153)
func (s *StateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
	if rules.IsEIP2929 && rules.IsEIP4762 {
		panic("eip2929 and eip4762 are both activated")
	}
	if rules.IsEIP2929 {
		// Clear out any leftover from previous executions
		al := newAccessList()
		s.accessList = al

		al.AddAddress(sender)
		if dst != nil {
			al.AddAddress(*dst)
			// If it's a create-tx, the destination will be added inside evm.create
		}
		for _, addr := range precompiles {
			al.AddAddress(addr)
		}
		for _, el := range list {
			al.AddAddress(el.Address)
			for _, key := range el.StorageKeys {
				al.AddSlot(el.Address, key)
			}
		}
		if rules.IsShanghai { // EIP-3651: warm coinbase
			al.AddAddress(coinbase)
		}
	}
	// Reset transient storage at the beginning of transaction execution
	s.transientStorage = newTransientStorage()
}

func (s *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

func (s *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := s.getOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, value)
	}
}

// SetTransientState sets transient storage for a given account. It
// adds the change to the journal so that it can be rolled back
// to its previous value if there is a revert.
func (s *StateDB) SetTransientState(addr common.Address, key, value common.Hash) {
	prev := s.GetTransientState(addr, key)
	if prev == value {
		return
	}
	s.journal.append(transientStorageChange{
		account:  &addr,
		key:      key,
		prevalue: prev,
	})
	s.setTransientState(addr, key, value)
}

// setTransientState is a lower level setter for transient storage. It
// is called during a revert to prevent modifications to the journal.
func (s *StateDB) setTransientState(addr common.Address, key, value common.Hash) {
	s.transientStorage.Set(addr, key, value)
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
func (s *StateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return s.accessList.Contains(addr, slot)
}
