package engine

import (
	"encoding/json"
	"fmt"
)

// ProtocolVersion is the JSON-Lines protocol version.
const ProtocolVersion = 1

// EngineVersion is reported by hello. It is a variable so release builds can
// inject the package version with -ldflags "-X mcmodlist/engine.EngineVersion=...".
var EngineVersion = "dev"

// Error codes are a closed set.
const (
	CodeInvalidRequest     = "invalid_request"
	CodeUnsupportedVersion = "unsupported_version"
	CodePathNotFound       = "path_not_found"
	CodeNotAnInstance      = "not_an_instance"
	CodeUnreadable         = "unreadable"
	CodeScanCancelled      = "scan_cancelled"
	CodeUnknownScan        = "unknown_scan"
	CodeInternal           = "internal"
)

// Methods is the supported method list, in documentation order.
var Methods = []string{
	"hello", "listInstances", "inspectPath", "scan", "cancel", "view",
	"details", "check", "compare", "export", "markdown", "shutdown",
}

// Request is one inbound JSON-Lines request.
type Request struct {
	V      int             `json:"v"`
	ID     uint64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// ErrorBody is the error payload for a response.
type ErrorBody struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Detail  json.RawMessage `json:"detail,omitempty"`
}

// Message is one outbound JSON-Lines response or event. A response sets Result
// or Error; an event sets Event and Data.
type Message struct {
	V      int             `json:"v"`
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorBody      `json:"error,omitempty"`
	Event  string          `json:"event,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

func response(id uint64, result any) Message {
	data, err := json.Marshal(result)
	if err != nil {
		return errorMessage(id, CodeInternal, err.Error(), nil)
	}
	return Message{V: ProtocolVersion, ID: id, Result: data}
}

func errorMessage(id uint64, code, message string, detail any) Message {
	var raw json.RawMessage
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			raw = b
		}
	}
	return Message{V: ProtocolVersion, ID: id, Error: &ErrorBody{Code: code, Message: message, Detail: raw}}
}

func event(id uint64, name string, data any) Message {
	raw, _ := json.Marshal(data)
	return Message{V: ProtocolVersion, ID: id, Event: name, Data: raw}
}

func decodeParams(req Request, out any) error {
	if len(req.Params) == 0 {
		return nil
	}
	if err := json.Unmarshal(req.Params, out); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}
