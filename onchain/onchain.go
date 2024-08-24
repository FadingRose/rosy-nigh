package onchain

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// impl Database for support of onchain fuzzing
// See [Database interface](../core/state/database.go)
type OnChainDataBase struct {
	apikeys   map[Chain]APIKey
	CodeCache map[common.Hash][]byte
}

func NewOnChainDataBase() *OnChainDataBase {
	return &OnChainDataBase{
		apikeys:   ApiKeys(),
		CodeCache: make(map[common.Hash][]byte),
	}
}

type ChainImpl interface {
	GetCode(address string, api APIKey) (string, error)
	GetCodeSize(address string, api APIKey) (int, error)
}

func (c *OnChainDataBase) ContractCode(address common.Address, hash common.Hash) ([]byte, error) {
	if code, ok := c.CodeCache[hash]; ok {
		return code, nil
	}
	// TODO support more chains
	// only support ETH chain for now
	eth := Chain(ETH)
	data, err := eth.GetCode(address.String(), c.apikeys[eth])
	if err != nil {
		return nil, err
	}
	hasher := crypto.NewKeccakState()
	hasher.Write(data)
	buf := common.Hash{}
	hasher.Read(buf[:])
	c.CodeCache[hash] = data
	return data, nil
}

func (c *OnChainDataBase) ContractCodeSize(address common.Address, hash common.Hash) (int, error) {
	if code, ok := c.CodeCache[hash]; ok {
		return len(code), nil
	}
	// TODO support more chains
	// only support ETH for now
	eth := Chain(ETH)
	data, err := eth.GetCode(address.String(), c.apikeys[eth])
	if err != nil {
		return 0, err
	}
	c.CodeCache[hash] = data
	return len(data), nil
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

func (c Chain) GetCode(address string, api APIKey) ([]byte, error) {
	args := map[string]string{
		"ADDRESS": address,
		"API_KEY": api,
	}
	endpoint := c.endpoint(callcode, args)
	return c.get(endpoint)
}

func (c Chain) get(endpoint string) ([]byte, error) {
	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	if err != nil {
		panic(err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * 10,
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type sample struct {
		Jsonrpc string `json:"jsonrpc"`
		Id      int    `json:"id"`
		Result  string `json:"result"`
	}
	var s sample
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return []byte(s.Result), nil
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
