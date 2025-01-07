package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

// StdioTransport implements Transport interface using stdin/stdout
type StdioTransport[T ID] struct {
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

func NewStdioTransport[T ID](opts ...TransportOption) *StdioTransport[T] {
	options := TransportOptions{}
	for _, apply := range opts {
		apply(&options)
	}
	return &StdioTransport[T]{
		options: options,
		reader:  bufio.NewReader(os.Stdin),
		writer:  bufio.NewWriter(os.Stdout),
	}
}

func (s *StdioTransport[T]) Start(ctx context.Context) error {
	if !s.state.CompareAndSwap(int32(stateStopped), int32(stateRunning)) {
		return ErrTransportStarted
	}

	s.outgoing = make(chan JSONRPCMessage, 100)
	s.incoming = make(chan JSONRPCMessage, 100)
	s.done = make(chan struct{})

	readReady := make(chan struct{})
	writeReady := make(chan struct{})

	// Start the read loop
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		close(readReady)
		s.readLoop(ctx)
	}()

	// Start the write loop
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		close(writeReady)
		s.writeLoop(ctx)
	}()

	// Wait for loops to be "ready"
	<-readReady
	<-writeReady

	return nil
}

func (s *StdioTransport[T]) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case msg := <-s.outgoing:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			data = append(data, '\n')

			if _, err := s.writer.Write(data); err != nil {
				fmt.Fprintf(os.Stderr, "write error: %v\n", err)
				continue
			}
			if err := s.writer.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "flush error: %v\n", err)
				continue
			}
		}
	}
}

// readLoop continuously reads from stdin and puts messages on incoming channel
func (s *StdioTransport[T]) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		default:
			line, err := s.reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF || s.state.Load() == int32(stateStopped) {
					select {
					case s.incoming <- &JSONRPCError[T]{
						Version: JSONRPCVersion,
						Err: Error{
							Code:    JSONRPCConnectionClosed,
							Message: ErrTransportClosed.Error(),
						},
					}:
					case <-s.done:
					case <-ctx.Done():
					}
					return
				}
				fmt.Fprintf(os.Stderr, "read error: %v\n", err)
				continue
			}

			// Remove trailing newline
			line = line[:len(line)-1]

			msg, err := parseJSONRPCMessage[T](line)
			if err != nil {
				msg = &JSONRPCError[T]{
					Version: JSONRPCVersion,
					Err: Error{
						Code:    JSONRPCParseError,
						Message: err.Error(),
					},
				}
			}

			select {
			case s.incoming <- msg:
			case <-ctx.Done():
				return
			case <-s.done:
				return
			}
		}
	}
}

func (s *StdioTransport[T]) Send(ctx context.Context, msg JSONRPCMessage) error {
	if s.state.Load() != int32(stateRunning) {
		return ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return ErrTransportClosed
	case s.outgoing <- msg:
		return nil
	}
}

func (s *StdioTransport[T]) Receive(ctx context.Context) (JSONRPCMessage, error) {
	if s.state.Load() != int32(stateRunning) {
		return nil, ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.done:
		return nil, ErrTransportClosed
	case msg := <-s.incoming:
		if msg.JSONRPCMessageType() == JSONRPCErrorMsgType {
			errMsg, ok := msg.(*JSONRPCError[T])
			if ok {
				if errMsg.Err.Code == JSONRPCConnectionClosed {
					if err := s.Close(); err != nil {
						return nil, fmt.Errorf("transport close: %v", err)
					}
					return nil, ErrTransportClosed
				}
			}
		}
		return msg, nil
	}
}

func (s *StdioTransport[T]) Close() error {
	if !s.state.CompareAndSwap(int32(stateRunning), int32(stateStopped)) {
		return nil
	}

	close(s.done)

	s.wg.Wait()

	close(s.incoming)
	close(s.outgoing)

	return s.writer.Flush()
}

// parseJSONRPCMessage attempts to parse a JSON-RPC message from raw bytes
func parseJSONRPCMessage[T ID](data []byte) (JSONRPCMessage, error) {
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
			var msg JSONRPCError[T]
			if err := json.Unmarshal(data, &msg); err != nil {
				return nil, fmt.Errorf("invalid JSON-RPC error: %w", err)
			}
			return &msg, nil
		}

		if len(base.Method) > 0 {
			var msg JSONRPCRequest[T]
			if err := json.Unmarshal(data, &msg); err != nil {
				return nil, fmt.Errorf("invalid JSON-RPC request: %w", err)
			}
			return &msg, nil
		}

		var msg JSONRPCResponse[T]
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC response: %w", err)
		}
		return &msg, nil
	}

	if len(base.Method) > 0 {
		var msg JSONRPCNotification[T]
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid JSON-RPC notification: %w", err)
		}
		return &msg, nil
	}

	return nil, ErrInvalidMessage
}
