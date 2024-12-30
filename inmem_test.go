package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestInMemTransport_Send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		tr := NewInMemTransport()
		msg := &JSONRPCRequest[uint64]{Version: JSONRPCVersion}
		err := tr.Send(context.Background(), msg)
		if err != nil {
			t.Errorf("Send() error = %v", err)
		}
	})

	t.Run("send after close", func(t *testing.T) {
		tr := NewInMemTransport()
		if err := tr.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		err := tr.Send(context.Background(), nil)
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Send() error = %v, want %v", err, ErrTransportClosed)
		}
	})

	t.Run("send with canceled context", func(t *testing.T) {
		tr := NewInMemTransport()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := tr.Send(ctx, nil)
		if err != context.Canceled {
			t.Errorf("Send() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestInMemTransport_Receive(t *testing.T) {
	t.Run("successful receive", func(t *testing.T) {
		tr := NewInMemTransport()
		sent := &JSONRPCRequest[uint64]{Version: JSONRPCVersion}
		go func() {
			_ = tr.Send(context.Background(), sent)
		}()

		received, err := tr.Receive(context.Background())
		if err != nil {
			t.Errorf("Receive() error = %v", err)
		}
		if !reflect.DeepEqual(received, sent) {
			t.Errorf("Receive() message = %v, want %v", received, sent)
		}
	})

	t.Run("receive after close", func(t *testing.T) {
		tr := NewInMemTransport()
		if err := tr.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		_, err := tr.Receive(context.Background())
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Receive() error = %v, want %v", err, ErrTransportClosed)
		}
	})

	t.Run("receive with canceled context", func(t *testing.T) {
		tr := NewInMemTransport()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := tr.Receive(ctx)
		if err != context.Canceled {
			t.Errorf("Receive() error = %v, want %v", err, context.Canceled)
		}
	})
}

func TestInMemTransport_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		tr := NewInMemTransport()
		if err := tr.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("multiple close", func(t *testing.T) {
		tr := NewInMemTransport()
		if err := tr.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if err := tr.Close(); err != nil {
			t.Errorf("Second Close() error = %v", err)
		}
	})
}

func TestInMemTransport_SendReceive(t *testing.T) {
	t.Run("send receive with timeout", func(t *testing.T) {
		tr := NewInMemTransport()
		msg := &JSONRPCRequest[uint64]{Version: JSONRPCVersion}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			_, err := tr.Receive(ctx)
			errCh <- err
		}()

		err := tr.Send(ctx, msg)
		if err != nil {
			t.Errorf("Send() error = %v", err)
		}

		if err := <-errCh; err != nil {
			t.Errorf("Receive() error = %v", err)
		}
	})

	t.Run("concurrent operations", func(t *testing.T) {
		tr := NewInMemTransport()

		const count = 10
		errCh := make(chan error, count*2)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		for range count {
			go func() {
				errCh <- tr.Send(ctx, &JSONRPCRequest[uint64]{})
			}()
			go func() {
				_, err := tr.Receive(ctx)
				errCh <- err
			}()
		}

		for range count * 2 {
			if err := <-errCh; err != nil {
				t.Errorf("Operation error = %v", err)
			}
		}
	})
}
