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
	return nil, nil
}

func (c *OnChainDataBase) ContractCodeSize(address common.Address, hash common.Hash) (int, error) {
	if code, ok := c.CodeCache[hash]; ok {
		return len(code), nil
	}
	return 0, nil
}

type Chain int

const (
	None = iota
	ETH
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

func StringToChain(s string) Chain {
	switch s {
	case "eth":
		return ETH
	case "goerli":
		return GOERLI
	case "sepolia":
		return SEPOLIA
	case "bsc":
		return BSC
	case "chapel":
		return CHAPEL
	case "polygon":
		return POLYGON
	case "mumbai":
		return MUMBAI
	case "fantom":
		return FANTOM
	case "avalanche":
		return AVALANCHE
	case "optimism":
		return OPTIMISM
	case "arbitrum":
		return ARBITRUM
	case "gnosis":
		return GNOSIS
	case "base":
		return BASE
	case "celo":
		return CELO
	case "zkevm":
		return ZKEVM
	case "zkevm_testnet":
		return ZkevmTestnet
	case "blast":
		return BLAST
	case "linea":
		return LINEA
	case "local":
		return LOCAL
	case "iotex":
		return IOTEX
	case "scroll":
		return SCROLL
	default:
		return None
	}
}
