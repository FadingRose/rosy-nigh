package vm

import (
	"fadingrose/rosy-nigh/core/tracing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// ContractRef is a reference to the contract's backing object
type ContractRef interface {
	Address() common.Address
}

// AccountRef implements ContractRef.
//
// Account references are used during EVM initialisation and
// its primary use is to fetch addresses. Removing this object
// proves difficult because of the cached jump destinations which
// are fetched from the parent contract (i.e. the caller), which
// is a ContractRef.
type AccountRef common.Address

// Address casts AccountRef to an Address
func (ar AccountRef) Address() common.Address { return (common.Address)(ar) }

// Contract represents an ethereum contract in the state database. It contains
// the contract code, calling arguments. Contract implements ContractRef
type Contract struct {
	// CallerAddress is the result of the caller which initialised this
	// contract. However when the "call method" is delegated this value
	// needs to be initialised to that of the caller's caller.
	CallerAddress common.Address
	caller        ContractRef
	self          ContractRef
	// WARNING: we do NOT know how to use `bitvec` and how it makes effects, ignore for now
	jumpdests map[common.Hash]bitvec // Aggregated result of JUMPDEST analysis.
	// analysis  bitvec                 // Locally cached result of JUMPDEST analysis

	Code     []byte
	CodeHash common.Hash
	CodeAddr *common.Address
	Input    []byte

	// is the execution frame represented by this object a contract deployment
	IsDeployment bool

	Gas   uint64
	value *uint256.Int
}

// NewContract returns a new contract environment for the execution of EVM.
func NewContract(caller ContractRef, object ContractRef, value *uint256.Int, gas uint64) *Contract {
	c := &Contract{CallerAddress: caller.Address(), caller: caller, self: object}

	if parent, ok := caller.(*Contract); ok {
		// Reuse JUMPDEST analysis from parent context if available.
		c.jumpdests = parent.jumpdests
	} else {
		c.jumpdests = make(map[common.Hash]bitvec)
	}

	// Gas should be a pointer so it can safely be reduced through the run
	// This pointer will be off the state transition
	c.Gas = gas
	// ensures a value is set
	c.value = value

	return c
}

// UseGas attempts the use gas and subtracts it and returns true on success
func (c *Contract) UseGas(gas uint64, logger *tracing.Hooks, reason tracing.GasChangeReason) (ok bool) {
	if c.Gas < gas {
		return false
	}
	if logger != nil && logger.OnGasChange != nil && reason != tracing.GasChangeIgnored {
		logger.OnGasChange(c.Gas, c.Gas-gas, reason)
	}
	c.Gas -= gas
	return true
}

// Address returns the contracts address
func (c *Contract) Address() common.Address {
	return c.self.Address()
}

// SetCodeOptionalHash can be used to provide code, but it's optional to provide hash.
// In case hash is not provided, the jumpdest analysis will not be saved to the parent context
func (c *Contract) SetCodeOptionalHash(addr *common.Address, codeAndHash *codeAndHash) {
	c.Code = codeAndHash.code
	c.CodeHash = codeAndHash.hash
	c.CodeAddr = addr
}

// Caller returns the caller of the contract.
//
// Caller will recursively call caller when the contract is a delegate
// call, including that of caller's caller.
func (c *Contract) Caller() common.Address {
	return c.CallerAddress
}

// Value returns the contract's value (sent to it from it's caller)
func (c *Contract) Value() *uint256.Int {
	return c.value
}

// GetOp returns the n'th element in the contract's byte array
func (c *Contract) GetOp(n uint64) OpCode {
	if n < uint64(len(c.Code)) {
		return OpCode(c.Code[n])
	}

	return STOP
}
