package dap

// https://microsoft.github.io/debug-adapter-protocol//specification.html
type DAPBaseMessage struct {
	Seq     uint64 `json:"seq"`
	Msgtype string `json:"type"`
}

type DAPRequest struct {
	DAPBaseMessage `json:",inline"`
	Command        string      `json:"command"`
	Arguments      interface{} `json:"arguments"`
}

type DAPEvent struct {
	DAPBaseMessage `json:",inline"`
	Event          string      `json:"event"`
	Body           interface{} `json:"body"`
}

type DAPResponse struct {
	DAPBaseMessage `json:",inline"`
	RequestSeq     int         `json:"request_seq"`
	Success        bool        `json:"success"`
	Message        string      `json:"message"` // 'cancelled' | 'notStopped' | string
	Body           interface{} `json:"body"`
}

type DAPErrorResponse struct {
	Body struct {
		Error string `json:"error"`
	}
}
