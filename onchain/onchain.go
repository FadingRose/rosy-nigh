package onchain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// impl Database for support of onchain fuzzing
// See [Database interface](../core/state/database.go)
type OnChainDataBase struct {
	apikeys   map[Chain]APIKey
	CodeCache map[common.Address][]byte
}

func NewOnChainDataBase() *OnChainDataBase {
	return &OnChainDataBase{
		apikeys:   ApiKeys(),
		CodeCache: loadCodeCache(),
	}
}

type ChainImpl interface {
	GetCode(address string, api APIKey) (string, error)
	GetCodeSize(address string, api APIKey) (int, error)
}

func (c *OnChainDataBase) ContractCode(address common.Address, hash common.Hash) ([]byte, error) {
	if code, ok := c.CodeCache[address]; ok {
		return code, nil
	}
	// TODO support more chains
	// only support ETH chain for now
	eth := Chain(ETH)
	data, err := eth.GetCode(address.String(), c.apikeys[eth])
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(string(data), "0x") {
		data = data[2:]
	}
	hash = hasher(data)
	c.CodeCache[address] = data

	appendCodeCache(address, data)

	return data, nil
}

func (c *OnChainDataBase) ContractCodeSize(address common.Address, hash common.Hash) (int, error) {
	if code, ok := c.CodeCache[address]; ok {
		return len(code), nil
	}
	eth := Chain(ETH)
	data, err := eth.GetCode(address.String(), c.apikeys[eth])
	if err != nil {
		return 0, err
	}
	if strings.HasPrefix(string(data), "0x") {
		data = data[2:]
	}
	hash = hasher(data)
	c.CodeCache[address] = data

	appendCodeCache(address, data)

	return len(data), nil
}

func hasher(data []byte) common.Hash {
	hasher := crypto.NewKeccakState()
	hasher.Write(data)
	buf := common.Hash{}
	hasher.Read(buf[:])
	return buf
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
	return c.getCode(endpoint)
}

func (c Chain) getCode(endpoint string) ([]byte, error) {
	type sample struct {
		Jsonrpc string `json:"jsonrpc"`
		Id      int    `json:"id"`
		Result  string `json:"result"`
	}
	var s sample
	s, err := get(endpoint, s)
	if err != nil {
		return nil, err
	}
	return []byte(s.Result), nil
}

func (c Chain) GetTx(txhash string, api APIKey) (from common.Address, input []byte, to common.Address, err error) {
	args := map[string]string{
		"TX_HASH": txhash,
		"API_KEY": api,
	}
	endpoint := c.endpoint(getTx, args)
	return c.getTx(endpoint)
}

func (c Chain) getTx(endpoint string) (from common.Address, input []byte, to common.Address, err error) {
	type sample struct {
		Jsonrpc string                 `json:"jsonrpc"`
		Id      int                    `json:"id"`
		Result  map[string]interface{} `json:"result"`
	}
	var s sample
	s, err = get(endpoint, s)
	if err != nil {
		return
	}

	if _, ok := s.Result["from"].(string); !ok {
		from = common.Address{}
	} else {
		from = common.HexToAddress(s.Result["from"].(string))
	}
	if _, ok := s.Result["input"].(string); !ok {
		input = []byte{}
	} else {
		inputStr := s.Result["input"].(string)
		inputStr = strings.TrimPrefix(inputStr, "0x")
		input = []byte(inputStr)
	}
	if _, ok := s.Result["to"].(string); !ok {
		to = common.Address{}
	} else {
		to = common.HexToAddress(s.Result["to"].(string))
	}
	return
}

func (c Chain) GetCreation(contractAddress string, api APIKey) (creator common.Address, txHash string, err error) {
	args := map[string]string{
		"CONTRACT_ADDRESS": contractAddress,
		"API_KEY":          api,
	}
	endpoint := c.endpoint(getcreation, args)
	return c.getCreation(endpoint)
}

func (c Chain) getCreation(endpoint string) (creator common.Address, txHash string, err error) {
	type sample struct {
		Jsonrpc string                   `json:"jsonrpc"`
		Id      int                      `json:"id"`
		Result  []map[string]interface{} `json:"result"`
	}
	var s sample
	s, err = get(endpoint, s)
	if err != nil {
		return
	}

	if _, ok := s.Result[0]["contractCreator"].(string); !ok {
		creator = common.Address{}
	} else {
		creator = common.HexToAddress(s.Result[0]["contractCreator"].(string))
	}

	if _, ok := s.Result[0]["txHash"].(string); !ok {
		txHash = ""
	} else {
		txHash = s.Result[0]["txHash"].(string)
	}

	return
}

func (c Chain) GetABI(contractAddress string, api APIKey) (abi string, err error) {
	args := map[string]string{
		"CONTRACT_ADDRESS": contractAddress,
		"API_KEY":          api,
	}
	endpoint := c.endpoint(getabi, args)
	return c.getABI(endpoint)
}

func (c Chain) getABI(endpoint string) (abi string, err error) {
	type sample struct {
		Jsonrpc string      `json:"jsonrpc"`
		Id      int         `json:"id"`
		Result  interface{} `json:"result"`
	}
	var s sample
	s, err = get(endpoint, s)
	if err != nil {
		return
	}
	if _, ok := s.Result.(string); !ok {
		abi = ""
	} else {
		abi = s.Result.(string)
	}
	return
}

func get[T any](endpoint string, s T) (T, error) {
	proxyURL, err := url.Parse("http://192.168.1.158:7890")
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
	fmt.Println("GET", endpoint)
	resp, err := client.Get(endpoint)
	if err != nil {
		return s, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return s, err
	}

	return s, nil
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
