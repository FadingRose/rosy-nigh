package onchain

import "strings"

func (c Chain) endpoint(method remoteCall, args map[string]string) string {
	return c.endpointBase() + remoteCalls()[c][method].impl(args)
}

func (c Chain) endpointBase() string {
	switch c {
	case ETH:
		return "https://api.etherscan.io/api"
	default:
		return ""
	}
}

type remoteCall int

const (
	callcode = iota
)

type remoteCallTmeplate string

func (rct remoteCallTmeplate) impl(args map[string]string) string {
	ret := string(rct)
	for k, v := range args {
		ret = strings.ReplaceAll(ret, "<"+k+">", v)
	}
	return ret
}

func remoteCalls() map[Chain]map[remoteCall]remoteCallTmeplate {
	return map[Chain]map[remoteCall]remoteCallTmeplate{
		ETH: {
			callcode: "?module=proxy&action=eth_getCode&address=<ADDRESS>&tag=latest&apikey=<API_KEY>",
		},
	}
}
