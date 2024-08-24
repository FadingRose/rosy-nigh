package onchain

import (
	"fmt"
	"io"
	"os"

	"github.com/pelletier/go-toml"
)

type APIKey = string

// ApiKeys returns a map of API keys for each chain
// Load keys from keys.toml
func ApiKeys() map[Chain]APIKey {
	keys, err := apikeysFromFile()
	if err != nil {
		fmt.Println("warning: failed to open keys.toml, online fuzzing disabled")
		return make(map[Chain]APIKey)
	}
	var config interface{}
	err = toml.Unmarshal(keys, &config)
	if err != nil {
		fmt.Println("warning: failed to unmarshal keys.toml, online fuzzing disabled")
		return make(map[Chain]APIKey)
	}
	m := config.(map[string]interface{})
	ret := make(map[Chain]APIKey)
	for k, v := range m {
		if StringToChain(k) == None {
			continue
		}
		ret[StringToChain(k)] = v.(string)
	}
	return ret
}

func apikeysFromFile() ([]byte, error) {
	// recursively search for keys.toml
	f, err := os.Open("keys.toml")
	if err != nil {
		return nil, fmt.Errorf("failed to open keys.toml: %w", err)
	}
	defer f.Close()
	return io.ReadAll(f)
}
