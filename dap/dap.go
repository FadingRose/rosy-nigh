package dap

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func sendDAPRequest(url string, request DAPRequest) (DAPResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return DAPResponse{}, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return DAPResponse{}, err
	}
	defer resp.Body.Close()

	var dapResponse DAPResponse
	err = json.NewDecoder(resp.Body).Decode(&dapResponse)
	if err != nil {
		return DAPResponse{}, err
	}

	return dapResponse, nil
}
