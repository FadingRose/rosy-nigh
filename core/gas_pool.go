package core

import "math"

// GasPool tracks the amount of gas available during execution of the transactions
// in a block. The zero value is a pool with zero gas available.
type GasPool uint64

// SubGas deducts the given amount from the pool if enough gas is
// available and returns an error otherwise.
// HACKED: Gas Pool now is UNLIMITED
func (gp *GasPool) SubGas(amount uint64) error {
	// if uint64(*gp) < amount {
	// 	return ErrGasLimitReached
	// }
	//*(*uint64)(gp) -= amount
	return nil
}

// AddGas makes gas available for execution.
func (gp *GasPool) AddGas(amount uint64) *GasPool {
	if uint64(*gp) > math.MaxUint64-amount {
		panic("gas pool pushed above uint64")
	}
	*(*uint64)(gp) += amount
	return gp
}
