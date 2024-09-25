package dap

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func startMockDAPServer() {
	http.HandleFunc("/dap", dapHandler)
	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Error:", err)
	}
}

func dapHandler(w http.ResponseWriter, r *http.Request) {
	var request DAPRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := DAPResponse{
		DAPBaseMessage: DAPBaseMessage{
			Seq:     request.Seq + 1,
			Msgtype: "response",
		},
		RequestSeq: 1,
		Success:    true,
		Body: map[string]interface{}{
			"stack": []map[string]interface{}{
				{
					"name":   "main",
					"line":   10,
					"column": 5,
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
