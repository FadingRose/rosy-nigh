package dap

import (
	"testing"
	"time"
)

func TestDAPStart(t *testing.T) {
	go startMockDAPServer()

	time.Sleep(1 * time.Second) // Wait for the server to start

	url := "http://localhost:8080/dap"
	request := DAPRequest{
		DAPBaseMessage: DAPBaseMessage{
			Seq:     1,
			Msgtype: "request",
		},
		Command: "stepIn",
		Arguments: map[string]int{
			"threadId": 1,
		},
	}

	response, err := sendDAPRequest(url, request)
	if err != nil {
		t.Errorf("Error: %v", err)
	} else {
		t.Logf("Response: %+v\n", response)
	}
}
