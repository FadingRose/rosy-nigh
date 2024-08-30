package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// ScopeContext contains the things that are per-call, such as stack and memory,
// but not transients like pc and gas
type ScopeContext struct {
	Memory   *Memory
	Stack    *Stack
	Contract *Contract
}

// impl SymbolicScope interface for symbolic execution

func (cts *ScopeContext) GetCaller() *uint256.Int {
	return uint256.NewInt(uint64(0)).SetBytes(cts.Caller().Bytes())
}

func (cts *ScopeContext) CallValue() *uint256.Int {
	return cts.Contract.Value()
}

func (cts *ScopeContext) GetData(data []byte, offset uint64, size uint64) []byte {
	return data[offset : offset+size]
}

func (cts *ScopeContext) GetInput() []byte {
	return cts.Contract.Input
}

func (cts *ScopeContext) CodeSize() *uint256.Int {
	return uint256.NewInt(uint64(len(cts.Contract.Code)))
}

func (cts *ScopeContext) GetCode() []byte {
	return cts.Contract.Code
}

func (cts *ScopeContext) GetGas() *uint256.Int {
	return uint256.NewInt(cts.Contract.Gas)
}

func (cts *ScopeContext) GetAddress() common.Address {
	return cts.Contract.Address()
}

func (cts *ScopeContext) MemorySize() int {
	return cts.Memory.Len()
}

// MemoryData returns the underlying memory slice. Callers must not modify the contents
// of the returned data.
func (ctx *ScopeContext) MemoryData() []byte {
	if ctx.Memory == nil {
		return nil
	}
	return ctx.Memory.Data()
}

// StackData returns the stack data. Callers must not modify the contents
// of the returned data.
func (ctx *ScopeContext) StackData() []uint256.Int {
	if ctx.Stack == nil {
		return nil
	}
	return ctx.Stack.Data()
}

// Caller returns the current caller.
func (ctx *ScopeContext) Caller() common.Address {
	return ctx.Contract.Caller()
}

// Address returns the address where this scope of execution is taking place.
func (ctx *ScopeContext) Address() common.Address {
	return ctx.Contract.Address()
}

// // CallValue returns the value supplied with this call.
// func (ctx *ScopeContext) CallValue() *uint256.Int {
// 	return ctx.Contract.Value()
// }

// CallInput returns the input/calldata with this call. Callers must not modify
// the contents of the returned data.
func (ctx *ScopeContext) CallInput() []byte {
	return ctx.Contract.Input
}
