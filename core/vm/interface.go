package vm

import "github.com/ethereum/go-ethereum/common"

type StateDB interface {
	CreateAccount(addr common.Address)
	CreateContract(addr common.Address, code []byte)
}
