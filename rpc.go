package mcp

import (
	"encoding/json"
	"fmt"
)

const (
	JSONRPCVersion = "2.0"
	LatestVersion  = "2024-11-05"
)

type JSONRPCMessageType string

const (
	JSONRPCRequestMsgType      JSONRPCMessageType = "request"
	JSONRPCNotificationMsgType JSONRPCMessageType = "notification"
	JSONRPCResponseMsgType     JSONRPCMessageType = "response"
	JSONRPCErrorMsgType        JSONRPCMessageType = "error"
)

// JSONRPCMessage is the interface for all JSON-RPC message types.
// JSONRPCRequest | JSONRPCNotification | JSONRPCResponse | JSONRPCError
type JSONRPCMessage interface {
	MessageType() JSONRPCMessageType
}

// DecodeJSONRPCMessage attempts to Unmarshal JSONRPCMessage from JSON and raturns it.
func DecodeJSONRPCMessage[T JSONRPCMessage](data []byte) (T, error) {
	var msg T
	if err := json.Unmarshal(data, &msg); err != nil {
		return msg, fmt.Errorf("failed to decode JSONRPCMessage: %w", err)
	}
	return msg, nil
}

// A request that expects a response.
type JSONRPCRequest struct {
	Request
	// ID corresponds to the JSON schema field "id".
	ID RequestID `json:"id"`
	// Jsonrpc corresponds to the JSON schema field "jsonrpc".
	// It must be set to JSONRPCVersion
	Jsonrpc string `json:"jsonrpc"`
}

// Implement JSONRPCMessage
func (j JSONRPCRequest) MessageType() JSONRPCMessageType {
	return JSONRPCRequestMsgType
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONRPCRequest) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["id"]; raw != nil && !ok {
		return fmt.Errorf("field id in JSONRPCRequest: required")
	}
	val, ok := raw["jsonrpc"]
	if raw != nil && !ok {
		return fmt.Errorf("field jsonrpc in JSONRPCRequest: required")
	}
	if strVal, ok := val.(string); !ok || strVal != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc in JSONRPCRequest: %v", val)
	}
	if _, ok := raw["method"]; raw != nil && !ok {
		return fmt.Errorf("field method in JSONRPCRequest: required")
	}
	type Plain JSONRPCRequest
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = JSONRPCRequest(plain)
	return nil
}

// A notification which does not expect a response.
type JSONRPCNotification struct {
	Notification
	// Jsonrpc corresponds to the JSON schema field "jsonrpc".
	Jsonrpc string `json:"jsonrpc"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONRPCNotification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["jsonrpc"]
	if raw != nil && !ok {
		return fmt.Errorf("field jsonrpc in JSONRPCRequest: required")
	}
	if strVal, ok := val.(string); !ok || strVal != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc in JSONRPCNotification: %v", val)
	}
	if _, ok := raw["method"]; raw != nil && !ok {
		return fmt.Errorf("field method in JSONRPCNotification: required")
	}
	type Plain JSONRPCNotification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = JSONRPCNotification(plain)
	return nil
}

// Implement JSONRPCMessage
func (j JSONRPCNotification) MessageType() JSONRPCMessageType {
	return JSONRPCNotificationMsgType
}

// A successful (non-error) response to a request.
type JSONRPCResponse struct {
	// ID corresponds to the JSON schema field "id".
	ID RequestID `json:"id"`
	// Jsonrpc corresponds to the JSON schema field "jsonrpc".
	Jsonrpc string `json:"jsonrpc"`
	// Result corresponds to the JSON schema field "result".
	Result Result `json:"result"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONRPCResponse) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["id"]; raw != nil && !ok {
		return fmt.Errorf("field id in JSONRPCResponse: required")
	}
	val, ok := raw["jsonrpc"]
	if raw != nil && !ok {
		return fmt.Errorf("field jsonrpc in JSONRPCRequest: required")
	}
	if strVal, ok := val.(string); !ok || strVal != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc in JSONRPCResponse: %v", val)
	}
	if _, ok := raw["result"]; raw != nil && !ok {
		return fmt.Errorf("field result in JSONRPCResponse: required")
	}
	type Plain JSONRPCResponse
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = JSONRPCResponse(plain)
	return nil
}

// Implement JSONRPCMessage
func (j JSONRPCResponse) MessageType() JSONRPCMessageType {
	return JSONRPCResponseMsgType
}

type JSONRPCErrorCode int

const (
	JSONRPCParseError          JSONRPCErrorCode = -32700
	JSONRPCInvalidRequestError JSONRPCErrorCode = -32600
	JSONRPCMethodNotFoundError JSONRPCErrorCode = -32601
	JSONRPCInvalidParamError   JSONRPCErrorCode = -32602
	JSONRPCInternalError       JSONRPCErrorCode = -32603
)

type JSONRPCErrorMsg struct {
	// The error type that occurred.
	Code JSONRPCErrorCode `json:"code"`
	// A short description of the error. The message SHOULD be limited to a concise
	// single sentence.
	Message string `json:"message"`
	// Additional information about the error. The value of this member is defined by
	// the sender (e.g. detailed error information, nested errors etc.).
	Data any `json:"data,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONRPCErrorMsg) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["code"]; raw != nil && !ok {
		return fmt.Errorf("field code in JSONRPCErrorError: required")
	}
	if _, ok := raw["message"]; raw != nil && !ok {
		return fmt.Errorf("field message in JSONRPCErrorError: required")
	}
	type Plain JSONRPCErrorMsg
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = JSONRPCErrorMsg(plain)
	return nil
}

// A response to a request that indicates an error occurred.
type JSONRPCError struct {
	// ID corresponds to the JSON schema field "id".
	ID RequestID `json:"id"`
	// Jsonrpc corresponds to the JSON schema field "jsonrpc".
	Jsonrpc string `json:"jsonrpc"`
	// Err corresponds to the JSON schema field "error".
	Err JSONRPCErrorMsg `json:"error"`
}

// Implement JSONRPCMessage
func (j JSONRPCError) MessageType() JSONRPCMessageType {
	return JSONRPCErrorMsgType
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONRPCError) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["error"]; raw != nil && !ok {
		return fmt.Errorf("field error in JSONRPCError: required")
	}
	if _, ok := raw["id"]; raw != nil && !ok {
		return fmt.Errorf("field id in JSONRPCError: required")
	}
	val, ok := raw["jsonrpc"]
	if raw != nil && !ok {
		return fmt.Errorf("field jsonrpc in JSONRPCRequest: required")
	}
	if strVal, ok := val.(string); !ok || strVal != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc in JSONRPCNotification: %v", val)
	}
	type Plain JSONRPCError
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = JSONRPCError(plain)
	return nil
}

func (j JSONRPCError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", j.Err.Code, j.Err.Message)
}
