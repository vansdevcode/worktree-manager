package daemon

import "encoding/json"

// Request is a JSON command sent to the daemon over the Unix socket.
type Request struct {
	Cmd        string `json:"cmd"`
	ConfigPath string `json:"config_path,omitempty"`
	WorkDir    string `json:"work_dir,omitempty"` // caller's working directory (used by process-up)
	Name       string `json:"name,omitempty"`     // process name (used by process-logs)
}

// Response is a JSON response sent back from the daemon.
type Response struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// OKResponse creates a successful response with optional data.
func OKResponse(data any) Response {
	var raw json.RawMessage
	if data != nil {
		raw, _ = json.Marshal(data)
	}
	return Response{OK: true, Data: raw}
}

// ErrResponse creates an error response.
func ErrResponse(err error) Response {
	return Response{OK: false, Error: err.Error()}
}
