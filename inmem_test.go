package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func InMemTransportMustStart(ctx context.Context, t *testing.T) *InMemTransport {
	tr := NewInMemTransport()
	if err := tr.Start(ctx); err != nil {
		t.Errorf("Start() error = %v, want %v", err, context.Canceled)
	}
	return tr
}

func InMemTransportMustClose(_ context.Context, t *testing.T, tr *InMemTransport) {
	if err := tr.Close(); err != nil {
		t.Fatalf("failed closing transport: %v", err)
	}
}

func TestInMemTransport_Start(t *testing.T) {
	t.Run("successful start", func(t *testing.T) {
		tr := NewInMemTransport()
		ctx := context.Background()
		if err := tr.Start(ctx); err != nil {
			t.Errorf("Start() error = %v", err)
		}
		InMemTransportMustClose(ctx, t, tr)
	})

	t.Run("start with canceled context", func(t *testing.T) {
		tr := NewInMemTransport()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := tr.Start(ctx); err != context.Canceled {
			t.Errorf("Start() error = %v, want %v", err, context.Canceled)
		}
		InMemTransportMustClose(ctx, t, tr)
	})

	t.Run("double start", func(t *testing.T) {
		tr := NewInMemTransport()
		ctx := context.Background()
		if err := tr.Start(ctx); err != nil {
			t.Fatalf("First Start() error = %v", err)
		}
		if err := tr.Start(ctx); !errors.Is(err, ErrTransportStarted) {
			t.Errorf("Second Start() error = %v, want %v", err, ErrTransportStarted)
		}
		InMemTransportMustClose(ctx, t, tr)
	})
}

func TestInMemTransport_Send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		ctx := context.Background()
		tr := InMemTransportMustStart(ctx, t)
		msg := &JSONRPCRequest[uint64]{Version: JSONRPCVersion}
		err := tr.Send(ctx, msg)
		if err != nil {
			t.Errorf("Send() error = %v", err)
		}
		InMemTransportMustClose(ctx, t, tr)
	})

	t.Run("send before start", func(t *testing.T) {
		tr := NewInMemTransport()
		if err := tr.Send(context.Background(), nil); !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Send() error = %v, want '%v'", err, ErrTransportClosed)
		}
	})

	t.Run("send after close", func(t *testing.T) {
		ctx := context.Background()
		tr := InMemTransportMustStart(ctx, t)
		InMemTransportMustClose(ctx, t, tr)

		err := tr.Send(context.Background(), nil)
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Send() error = %v, want %v", err, ErrTransportClosed)
		}
	})

	t.Run("send with canceled context", func(t *testing.T) {
		tr := InMemTransportMustStart(context.Background(), t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := tr.Send(ctx, nil); err != context.Canceled {
			t.Errorf("Send() error = %v, want %v", err, context.Canceled)
		}
		InMemTransportMustClose(context.Background(), t, tr)
	})
}

func TestInMemTransport_Receive(t *testing.T) {
	t.Run("successful receive", func(t *testing.T) {
		ctx := context.Background()
		tr := InMemTransportMustStart(ctx, t)
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
		InMemTransportMustClose(ctx, t, tr)
	})

	t.Run("receive before start", func(t *testing.T) {
		tr := NewInMemTransport()
		if _, err := tr.Receive(context.Background()); !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Receive() error = %v, want '%v'", err, ErrTransportClosed)
		}
	})

	t.Run("receive after close", func(t *testing.T) {
		ctx := context.Background()
		tr := InMemTransportMustStart(ctx, t)
		if err := tr.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		_, err := tr.Receive(context.Background())
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Receive() error = %v, want %v", err, ErrTransportClosed)
		}
	})

	t.Run("receive with canceled context", func(t *testing.T) {
		ctx := context.Background()
		tr := InMemTransportMustStart(ctx, t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := tr.Receive(ctx)
		if err != context.Canceled {
			t.Errorf("Receive() error = %v, want %v", err, context.Canceled)
		}
		InMemTransportMustClose(ctx, t, tr)
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
		ctx := context.Background()
		tr := InMemTransportMustStart(ctx, t)
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
		ctx := context.Background()
		tr := InMemTransportMustStart(ctx, t)
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
		InMemTransportMustClose(ctx, t, tr)
	})

	t.Run("concurrent operations", func(t *testing.T) {
		ctx := context.Background()
		tr := InMemTransportMustStart(ctx, t)

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
		InMemTransportMustClose(ctx, t, tr)
	})
}
