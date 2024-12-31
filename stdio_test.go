package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
)

func TestStdioTransport_Send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		os.Stdout = w

		tr := NewStdioTransport()

		msg := &JSONRPCRequest[uint64]{
			Request: &PingRequest[uint64]{
				Request: Request[uint64]{
					Method: PingRequestMethod,
				},
			},
			Version: JSONRPCVersion,
		}
		err = tr.Send(context.Background(), msg)
		if err != nil {
			t.Errorf("Send() error = %v", err)
		}

		if err := w.Close(); err != nil {
			t.Errorf("w.Close() error = %v", err)
		}

		// Restore stdout
		os.Stdout = oldStdout

		// Read the output and verify
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}

		// Verify the output is valid JSON followed by newline
		var received JSONRPCRequest[uint64]
		if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &received); err != nil {
			t.Errorf("Failed to unmarshal sent message: %v", err)
		}
		if received.Version != JSONRPCVersion {
			t.Errorf("Sent message version = %v, want %v", received.Version, JSONRPCVersion)
		}
	})

	t.Run("send after close", func(t *testing.T) {
		tr := NewStdioTransport()
		if err := tr.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		err := tr.Send(context.Background(), nil)
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Send() error = %v, want %v", err, ErrTransportClosed)
		}
	})

	t.Run("send with canceled context", func(t *testing.T) {
		tr := NewStdioTransport()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := tr.Send(ctx, nil)
		if err != context.Canceled {
			t.Errorf("Send() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestStdioTransport_Receive(t *testing.T) {
	t.Run("successful receive", func(t *testing.T) {
		// Replace stdin with a pipe
		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		os.Stdin = r

		// Write a test message to the pipe
		msg := &JSONRPCRequest[uint64]{
			Request: &PingRequest[uint64]{
				Request: Request[uint64]{
					Method: PingRequestMethod,
				},
			},
			Version: JSONRPCVersion,
		}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal message: %v", err)
		}
		data = append(data, '\n')
		if _, err := w.Write(data); err != nil {
			t.Fatalf("Failed to write to pipe: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Errorf("w.Close() error = %v", err)
		}

		tr := NewStdioTransport()
		received, err := tr.Receive(context.Background())
		if err != nil {
			t.Errorf("Receive() error = %v", err)
		}

		// Restore stdin
		os.Stdin = oldStdin

		// Verify received message
		req, ok := received.(*JSONRPCRequest[uint64])
		if !ok {
			t.Errorf("Received message type = %T, want *JSONRPCRequest[uint64]", received)
		}
		if req.Version != JSONRPCVersion {
			t.Errorf("Received message version = %v, want %v", req.Version, JSONRPCVersion)
		}
	})

	t.Run("receive after close", func(t *testing.T) {
		tr := NewStdioTransport()
		if err := tr.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		_, err := tr.Receive(context.Background())
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Receive() error = %v, want %v", err, ErrTransportClosed)
		}
	})

	t.Run("receive with canceled context", func(t *testing.T) {
		tr := NewStdioTransport()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := tr.Receive(ctx)
		if err != context.Canceled {
			t.Errorf("Receive() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestStdioTransport_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		tr := NewStdioTransport()
		if err := tr.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("multiple close", func(t *testing.T) {
		tr := NewStdioTransport()
		if err := tr.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if err := tr.Close(); err != nil {
			t.Errorf("Second Close() error = %v", err)
		}
	})
}
