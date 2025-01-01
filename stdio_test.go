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

func StdioTransportMustStart(ctx context.Context, t *testing.T) *StdioTransport {
	tr := NewStdioTransport()
	if err := tr.Start(ctx); err != nil {
		t.Errorf("Start() error = %v, want %v", err, context.Canceled)
	}
	return tr
}

func StdioTransportMustClose(_ context.Context, t *testing.T, tr *StdioTransport) {
	if err := tr.Close(); err != nil {
		t.Fatalf("failed closing transport: %v", err)
	}
}

func TestStdioTransport_Start(t *testing.T) {
	t.Run("successful start", func(t *testing.T) {
		tr := NewStdioTransport()
		if err := tr.Start(context.Background()); err != nil {
			t.Errorf("Start() error = %v", err)
		}
		StdioTransportMustClose(context.Background(), t, tr)
	})

	t.Run("double start", func(t *testing.T) {
		tr := NewStdioTransport()
		if err := tr.Start(context.Background()); err != nil {
			t.Fatalf("First Start() error = %v", err)
		}
		if err := tr.Start(context.Background()); !errors.Is(err, ErrTransportStarted) {
			t.Errorf("Second Start() error = %v, want %v", err, ErrTransportStarted)
		}
		StdioTransportMustClose(context.Background(), t, tr)
	})
}

func TestStdioTransport_Send(t *testing.T) {
	t.Run("successful_send", func(t *testing.T) {
		// 1) Replace stdout with a pipe.
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		os.Stdout = w

		// 2) Start transport
		ctx := context.Background()
		tr := StdioTransportMustStart(ctx, t)

		msg := &JSONRPCRequest[uint64]{
			Request: &PingRequest[uint64]{
				Request: Request[uint64]{
					Method: PingRequestMethod,
				},
			},
			Version: JSONRPCVersion,
		}
		if err := tr.Send(context.Background(), msg); err != nil {
			t.Errorf("Send() error = %v", err)
		}

		// 4) Close transport => blocks until writer goroutine is done
		if err := tr.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}

		// 5) Now close the pipe’s writer end
		if err := w.Close(); err != nil {
			t.Errorf("w.Close() error = %v", err)
		}

		// 6) Restore stdout
		os.Stdout = oldStdout

		// 7) Read all data from r
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			t.Fatalf("Failed to read from pipe: %v", err)
		}

		// 8) Verify JSON
		var received JSONRPCRequest[uint64]
		if err := json.Unmarshal(buf.Bytes(), &received); err != nil {
			t.Fatalf("Failed to unmarshal sent message: %v", err)
		}
		if received.Version != JSONRPCVersion {
			t.Errorf("Sent message version = %v, want %v", received.Version, JSONRPCVersion)
		}
	})

	t.Run("send before start", func(t *testing.T) {
		tr := NewStdioTransport()
		if err := tr.Send(context.Background(), nil); !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Send() error = %v, want '%v'", err, ErrTransportClosed)
		}
	})

	t.Run("send after close", func(t *testing.T) {
		ctx := context.Background()
		tr := StdioTransportMustStart(ctx, t)
		StdioTransportMustClose(context.Background(), t, tr)

		err := tr.Send(context.Background(), nil)
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Send() error = %v, want %v", err, ErrTransportClosed)
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

		ctx := context.Background()
		tr := StdioTransportMustStart(ctx, t)

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

		StdioTransportMustClose(context.Background(), t, tr)
	})

	t.Run("receive before start", func(t *testing.T) {
		tr := NewStdioTransport()
		if _, err := tr.Receive(context.Background()); !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Receive() error = %v, want '%v'", err, ErrTransportClosed)
		}
	})

	t.Run("receive after close", func(t *testing.T) {
		ctx := context.Background()
		tr := StdioTransportMustStart(ctx, t)
		if err := tr.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		_, err := tr.Receive(context.Background())
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Receive() error = %v, want %v", err, ErrTransportClosed)
		}
	})
}

func TestStdioTransport_Close(t *testing.T) {
	t.Run("close before start", func(t *testing.T) {
		tr := NewStdioTransport()
		if err := tr.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("successful close", func(t *testing.T) {
		ctx := context.Background()
		tr := StdioTransportMustStart(ctx, t)
		if err := tr.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("multiple close", func(t *testing.T) {
		ctx := context.Background()
		tr := StdioTransportMustStart(ctx, t)
		if err := tr.Close(); err != nil {
			t.Fatalf("First Close() error = %v", err)
		}
		if err := tr.Close(); err != nil {
			t.Errorf("Second Close() error = %v", err)
		}
	})
}
