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
)

// StdioTransport implements Transport interface using stdin/stdout
type StdioTransport struct {
	options TransportOptions
	reader  *bufio.Reader
	writer  *bufio.Writer

	// Channels for passing messages between I/O loops and Send/Receive methods
	outgoing chan JSONRPCMessage
	incoming chan JSONRPCMessage

	// Channel to signal shutdown
	done chan struct{}

	// WaitGroup to track background goroutines
	wg sync.WaitGroup

	state atomic.Int32
}

func NewStdioTransport(opts ...TransportOption) *StdioTransport {
	options := TransportOptions{}
	for _, apply := range opts {
		apply(&options)
	}
	return &StdioTransport{
		options: options,
		reader:  bufio.NewReader(os.Stdin),
		writer:  bufio.NewWriter(os.Stdout),
	}
}

func (t *StdioTransport) Start(ctx context.Context) error {
	if !t.state.CompareAndSwap(int32(stateStopped), int32(stateRunning)) {
		return ErrTransportStarted
	}

	t.outgoing = make(chan JSONRPCMessage, 100)
	t.incoming = make(chan JSONRPCMessage, 100)
	t.done = make(chan struct{})

	readReady := make(chan struct{})
	writeReady := make(chan struct{})

	// Start the read loop
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		close(readReady)
		t.readLoop(ctx)
	}()

	// Start the write loop
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		close(writeReady)
		t.writeLoop(ctx)
	}()

	// Wait for loops to be "ready"
	<-readReady
	<-writeReady

	return nil
}

func (t *StdioTransport) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.done:
			return
		case msg := <-t.outgoing:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			data = append(data, '\n')
			if _, err := t.writer.Write(data); err != nil {
				continue
			}
			if err := t.writer.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "flush error: %v\n", err)
				continue
			}
		}
	}
}

// readLoop continuously reads from stdin and puts messages on incoming channel
func (t *StdioTransport) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.done:
			return
		default:
			line, err := t.reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF || t.state.Load() == int32(stateStopped) {
					return
				}
				// Log error but continue trying
				fmt.Fprintf(os.Stderr, "stdio read error: %v\n", err)
				continue
			}

			// Remove trailing newline
			line = line[:len(line)-1]

			msg, err := parseJSONRPCMessage(line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "parse message error: %v\n", err)
				continue
			}

			select {
			case t.incoming <- msg:
			case <-ctx.Done():
				return
			case <-t.done:
				return
			}
		}
	}
}

func (t *StdioTransport) Send(ctx context.Context, msg JSONRPCMessage) error {
	if t.state.Load() != int32(stateRunning) {
		return ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.outgoing <- msg:
		return nil
	}
}

func (t *StdioTransport) Receive(ctx context.Context) (JSONRPCMessage, error) {
	if t.state.Load() != int32(stateRunning) {
		return nil, ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-t.incoming:
		return msg, nil
	}
}

func (t *StdioTransport) Close() error {
	if !t.state.CompareAndSwap(int32(stateRunning), int32(stateStopped)) {
		return nil
	}

	close(t.done)

	// Wait for loops to finish
	t.wg.Wait()

	// Close channels
	close(t.incoming)
	close(t.outgoing)

	// Final flush
	return t.writer.Flush()
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
