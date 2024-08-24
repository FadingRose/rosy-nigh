package onchain

import "testing"

func TestEndpoints(t *testing.T) {
	tcs := []struct {
		Chain
		remoteCall
		key      APIKey
		args     map[string]string
		expected string
	}{
		{
			Chain:      ETH,
			remoteCall: callcode,
			args: map[string]string{
				"ADDRESS": "0x123",
				"API_KEY": "acbdef",
			},
			expected: "https://api.etherscan.io/api?module=proxy&action=eth_Code&address=0x123&tag=latest&apikey=acbdef",
		},
	}
	for _, tc := range tcs {
		actual := tc.Chain.endpoint(tc.remoteCall, tc.args)
		if actual != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, actual)
		}
	}
}
