package types

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// LegacyTx is the transaction data of the original Ethereum transactions.
type LegacyTx struct {
	Nonce    uint64          // nonce of sender account
	GasPrice *big.Int        // wei per gas
	Gas      uint64          // gas limit
	To       *common.Address `rlp:"nil"` // nil means contract creation
	Value    *big.Int        // wei amount
	Data     []byte          // contract invocation input data
	V, R, S  *big.Int        // signature values
}

func NewTx(inner TxData) *Transaction {
	return &Transaction{
		inner: inner,
		time:  time.Now(),
	}
}

// accessors for innerTx.
func (tx *LegacyTx) txType() byte { return LegacyTxType }
