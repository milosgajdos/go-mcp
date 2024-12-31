package mcp

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Ensure StdioTransport implements Transport interface
var _ Transport = (*InMemTransport)(nil)

type InMemTransport struct {
	ch        chan JSONRPCMessage
	options   TransportOptions
	closeOnce sync.Once
	closed    atomic.Bool
}

func NewInMemTransport(opts ...TransportOption) *InMemTransport {
	options := TransportOptions{}
	for _, apply := range opts {
		apply(&options)
	}
	return &InMemTransport{
		options: options,
		ch:      make(chan JSONRPCMessage, 1),
	}
}

func (t *InMemTransport) Send(ctx context.Context, msg JSONRPCMessage) error {
	if t.closed.Load() {
		return fmt.Errorf("%w: inmem: %v", ErrTransportClosed, io.ErrClosedPipe)
	}

	// exit early if context is cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	if t.options.SendDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(t.options.RecvDelay): // Simulate latency
		}
	}

	if t.options.SendTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.options.SendTimeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.ch <- msg:
		return nil
	}
}

func (t *InMemTransport) Receive(ctx context.Context) (JSONRPCMessage, error) {
	if t.closed.Load() {
		return nil, fmt.Errorf("%w: inmem: %v", ErrTransportClosed, io.ErrClosedPipe)
	}
	// exit early if context is cancelled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if t.options.RecvDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(t.options.RecvDelay): // Simulate latency
		}
	}

	if t.options.RecvTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.options.RecvTimeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-t.ch:
		return msg, nil
	}
}

func (t *InMemTransport) Close() error {
	t.closeOnce.Do(func() {
		t.closed.Store(true)
		close(t.ch)
	})
	return nil
}
