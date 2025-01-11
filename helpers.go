package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
)

// NewProgressToken creates a new request ID and returns it.
func NewRequestID[T ~uint64 | ~string](id T) RequestID {
	return RequestID{
		Value: ID(id),
	}
}

// NewProgressToken creates a new progress token and returns it.
func NewProgressToken[T ~uint64 | ~string](id T) ProgressToken {
	return ProgressToken{
		Value: ID(id),
	}
}

// parseJSONRPCMessage attempts to parse a JSON-RPC message from raw bytes
func parseJSONRPCMessage(data []byte) (JSONRPCMessage, error) {
	// First unmarshal to get the basic structure
	var base struct {
		Version string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  json.RawMessage `json:"method"`
		Error   json.RawMessage `json:"error"`
	}

	if err := json.Unmarshal(data, &base); err != nil {
		return nil, errors.Join(ErrInvalidMessage, err)
	}

	// Determine message type based on fields
	if len(base.ID) > 0 {
		if len(base.Error) > 0 {
			var msg JSONRPCError
			if err := json.Unmarshal(data, &msg); err != nil {
				return nil, fmt.Errorf("invalid JSON-RPC error: %w", err)
			}
			return &msg, nil
		}

		if len(base.Method) > 0 {
			var msg JSONRPCRequest
			if err := json.Unmarshal(data, &msg); err != nil {
				return nil, fmt.Errorf("invalid JSON-RPC request: %w", err)
			}
			return &msg, nil
		}

		var msg JSONRPCResponse
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC response: %w", err)
		}
		return &msg, nil
	}

	if len(base.Method) > 0 {
		var msg JSONRPCNotification
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC notification: %w", err)
		}
		return &msg, nil
	}

	return nil, ErrInvalidMessage
}
