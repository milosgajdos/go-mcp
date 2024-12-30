package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Ensure StdioTransport implements Transport interface
var _ Transport = (*StdioTransport)(nil)

// StdioTransport implements Transport interface using stdin/stdout
type StdioTransport struct {
	options   TransportOptions
	reader    *bufio.Reader
	writer    *bufio.Writer
	writeMu   sync.Mutex
	closeOnce sync.Once
	closed    atomic.Bool
}

// NewStdioTransport creates a new stdio transport and returns it.
func NewStdioTransport(opts ...TransportOption) *StdioTransport {
	options := TransportOptions{}
	for _, apply := range opts {
		apply(&options)
	}
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}
}

// Send implements Transport interface
func (t *StdioTransport) Send(ctx context.Context, msg JSONRPCMessage) error {
	if t.closed.Load() {
		return fmt.Errorf("%w: stdio: %v", ErrTransportClosed, io.ErrClosedPipe)
	}

	if t.options.SendTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.options.SendTimeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		t.writeMu.Lock()
		defer t.writeMu.Unlock()

		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON-RPC message: %w", err)
		}
		// Add newline to separate messages
		data = append(data, '\n')

		if _, err := t.writer.Write(data); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
		if err := t.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush stdout: %w", err)
		}
		return nil
	}
}

// Receive implements Transport interface
func (t *StdioTransport) Receive(ctx context.Context) (JSONRPCMessage, error) {
	if t.closed.Load() {
		return nil, fmt.Errorf("%w: stdio: %v", ErrTransportClosed, io.ErrClosedPipe)
	}

	if t.options.RecvDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(t.options.RecvDelay): // Simulate latency
		}
	}

	// Check context before attempting read
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// ReadBytes is already buffered through bufio.Reader
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		// Remove trailing newline
		line = line[:len(line)-1]
		// Parse the message
		msg, err := parseJSONRPCMessage(line)
		if err != nil {
			return nil, err
		}

		return msg, nil
	}
}

// Close implements Transport interface
func (t *StdioTransport) Close() error {
	t.closeOnce.Do(func() {
		t.closed.Store(true)
		_ = t.writer.Flush()
	})
	return nil
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
		return nil, fmt.Errorf("invalid JSON-RPC message: %w", err)
	}

	// Determine message type based on fields
	if len(base.ID) > 0 {
		if len(base.Error) > 0 {
			var msg JSONRPCError[uint64]
			if err := json.Unmarshal(data, &msg); err != nil {
				return nil, fmt.Errorf("invalid JSON-RPC error: %w", err)
			}
			return &msg, nil
		}

		if len(base.Method) > 0 {
			var msg JSONRPCRequest[uint64]
			if err := json.Unmarshal(data, &msg); err != nil {
				return nil, fmt.Errorf("invalid JSON-RPC request: %w", err)
			}
			return &msg, nil
		}

		var msg JSONRPCResponse[uint64]
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC response: %w", err)
		}
		return &msg, nil
	}

	if len(base.Method) > 0 {
		var msg JSONRPCNotification[uint64]
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC notification: %w", err)
		}
		return &msg, nil
	}

	return nil, fmt.Errorf("invalid JSON-RPC message format")
}
