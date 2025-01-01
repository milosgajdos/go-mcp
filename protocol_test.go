package mcp

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProtocol_Connect(t *testing.T) {
	t.Run("Successful Connect", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !p.running.Load() {
			t.Fatal("Protocol not marked as running")
		}
	})

	t.Run("Already Connected", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Protocol failed to connect: %v", err)
		}
		if err := p.Connect(); err == nil || err != ErrAlreadyConnected {
			t.Fatalf("Expected already connected error, got: %v", err)
		}
	})

	t.Run("Invalid Transport", func(t *testing.T) {
		invalidProtocol := NewProtocol[uint64](WithTransport(nil)) // nil transport
		if err := invalidProtocol.Connect(); err == nil || err != ErrInvalidTransport {
			t.Fatalf("Expected invalid transport error, got: %v", err)
		}
	})
}

func TestProtocol_Close(t *testing.T) {
	t.Run("Successful Close", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Protocol failed to connect: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if p.running.Load() {
			t.Fatalf("Protocol not marked as stopped")
		}
	})

	t.Run("Close When Already Closed", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Protocol failed to connect: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("Expected no error on second close, got: %v", err)
		}
	})

	t.Run("Close With Pending Requests", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Protocol failed to connect: %v", err)
		}
		p.pendingMu.Lock()
		p.pending[RequestID[uint64]{Value: 1}] = make(chan RespOrError[uint64], 1)
		p.pendingMu.Unlock()

		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("Expected no error on close, got: %v", err)
		}
		if p.running.Load() {
			t.Fatalf("Protocol not marked as stopped")
		}
	})
}

func TestProtocol_SendRequest(t *testing.T) {
	t.Run("Successful Request Response", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](
			WithTransport(tr),
			WithReqTimeout(100*time.Millisecond),
		)

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		// Register handler for ping
		p.RegisterRequestHandler(PingRequestMethod,
			func(_ context.Context, req *JSONRPCRequest[uint64]) (*JSONRPCResponse[uint64], error) {
				return &JSONRPCResponse[uint64]{
					Result:  &PingResult{},
					ID:      req.ID,
					Version: JSONRPCVersion,
				}, nil
			})

		resp, err := p.SendRequest(context.Background(), &JSONRPCRequest[uint64]{
			Request: &PingRequest[uint64]{
				Request: Request[uint64]{
					Method: PingRequestMethod,
				},
			},
		})

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp == nil {
			t.Fatal("Expected response, got nil")
		}
		if _, ok := resp.Result.(*PingResult); !ok {
			t.Fatal("Expected PingResult")
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}
	})

	t.Run("Request Timeout", func(t *testing.T) {
		timeout := 50 * time.Millisecond
		tr := NewInMemTransport(WithRecvDelay(2 * timeout))
		p := NewProtocol[uint64](
			WithTransport(tr),
			WithReqTimeout(timeout),
		)

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		_, err := p.SendRequest(context.Background(), &JSONRPCRequest[uint64]{
			Request: &PingRequest[uint64]{
				Request: Request[uint64]{
					Method: PingRequestMethod,
				},
			},
		})

		var rpcErr *JSONRPCError[uint64]
		if !errors.As(err, &rpcErr) || rpcErr.Err.Code != JSONRPCRequestTimeout {
			t.Fatalf("Expected timeout error, got: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}
	})

	t.Run("Request With Canceled Context", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := p.SendRequest(ctx, &JSONRPCRequest[uint64]{
			Request: &PingRequest[uint64]{
				Request: Request[uint64]{
					Method: PingRequestMethod,
				},
			},
		})

		if err != context.Canceled {
			t.Fatalf("Expected context.Canceled, got: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}
	})

	t.Run("Request After Close", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}

		_, err := p.SendRequest(context.Background(), &JSONRPCRequest[uint64]{
			Request: &PingRequest[uint64]{
				Request: Request[uint64]{
					Method: PingRequestMethod,
				},
			},
		})

		if err == nil || !errors.Is(err, ErrTransportClosed) {
			t.Fatalf("Expected transport closed error, got: %v", err)
		}
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](
			WithTransport(tr),
			WithReqTimeout(100*time.Millisecond),
		)

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		// Register ping handler
		p.RegisterRequestHandler(PingRequestMethod,
			func(_ context.Context, req *JSONRPCRequest[uint64]) (*JSONRPCResponse[uint64], error) {
				return &JSONRPCResponse[uint64]{
					Result:  &PingResult{},
					ID:      req.ID,
					Version: JSONRPCVersion,
				}, nil
			})

		const numRequests = 10
		errs := make(chan error, numRequests)

		for range numRequests {
			go func() {
				_, err := p.SendRequest(context.Background(), &JSONRPCRequest[uint64]{
					Request: &PingRequest[uint64]{
						Request: Request[uint64]{
							Method: PingRequestMethod,
						},
					},
				})
				errs <- err
			}()
		}

		for i := range numRequests {
			if err := <-errs; err != nil {
				t.Errorf("Request %d failed: %v", i, err)
			}
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}
	})
}

func TestProtocol_SendNotification(t *testing.T) {
	t.Run("Successful Notification", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		err := p.SendNotification(context.Background(), &JSONRPCNotification[uint64]{
			Notification: &InitializedNotification{
				Notification: Notification{
					Method: InitializedNotificationMethod,
				},
			},
		})

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}
	})

	t.Run("Notification With Canceled Context", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := p.SendNotification(ctx, &JSONRPCNotification[uint64]{
			Notification: &InitializedNotification{
				Notification: Notification{
					Method: InitializedNotificationMethod,
				},
			},
		})

		if err != context.Canceled {
			t.Fatalf("Expected context.Canceled, got: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}
	})

	t.Run("Notification After Close", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}

		err := p.SendNotification(context.Background(), &JSONRPCNotification[uint64]{
			Notification: &InitializedNotification{
				Notification: Notification{
					Method: InitializedNotificationMethod,
				},
			},
		})

		if err == nil || !errors.Is(err, ErrTransportClosed) {
			t.Fatalf("Expected transport closed error, got: %v", err)
		}
	})

	t.Run("Concurrent Notifications", func(t *testing.T) {
		tr := NewInMemTransport()
		p := NewProtocol[uint64](WithTransport(tr))

		if err := p.Connect(); err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		const numNotifications = 10
		errs := make(chan error, numNotifications)

		for range numNotifications {
			go func() {
				err := p.SendNotification(context.Background(), &JSONRPCNotification[uint64]{
					Notification: &InitializedNotification{
						Notification: Notification{
							Method: InitializedNotificationMethod,
						},
					},
				})
				errs <- err
			}()
		}

		for i := range numNotifications {
			if err := <-errs; err != nil {
				t.Errorf("Notification %d failed: %v", i, err)
			}
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatalf("failed to close protocol: %v", err)
		}
	})
}
