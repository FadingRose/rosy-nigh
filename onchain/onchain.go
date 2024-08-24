package onchain

import (
	"net/http"

	"github.com/ethereum/go-ethereum/common"
)

// impl Database for support of onchain fuzzing
// See [Database interface](../core/state/database.go)

type OnChainDataBase struct {
	ChainImpls  map[Chain]ChainImpl
	EndpointURL string
	Client      *http.Client
	APIKey      map[Chain]API
	CodeCache   map[common.Hash][]byte
}

func NewOnChainDataBase() *OnChainDataBase {
	return &OnChainDataBase{
		ChainImpls: make(map[Chain]ChainImpl),
		Client:     &http.Client{},
		APIKey:     ApiKeys(),
		CodeCache:  make(map[common.Hash][]byte),
	}
}

type ChainImpl interface {
	GetCode(address string, api API) (string, error)
}

func (c *OnChainDataBase) ContractCode(address common.Address, hash common.Hash) ([]byte, error) {
	if code, ok := c.CodeCache[hash]; ok {
		return code, nil
	}
}

func (c *OnChainDataBase) ContractCodeSize(address common.Address, hash common.Hash) (int, error) {
	if code, ok := c.CodeCache[hash]; ok {
		return len(code), nil
	}
}

type Chain int

const (
	ETH = iota
	GOERLI
	SEPOLIA
	BSC
	CHAPEL
	POLYGON
	MUMBAI
	FANTOM
	AVALANCHE
	OPTIMISM
	ARBITRUM
	GNOSIS
	BASE
	CELO
	ZKEVM
	ZkevmTestnet
	BLAST
	LINEA
	LOCAL
	IOTEX
	SCROLL
)

func (c Chain) String() string {
	return [...]string{"eth", "goerli", "sepolia", "bsc", "chapel", "polygon", "mumbai", "fantom", "avalanche", "optimism", "arbitrum", "gnosis", "base", "celo", "zkevm", "zkevm_testnet", "blast", "linea", "local", "iotex", "scroll"}[c]
}
