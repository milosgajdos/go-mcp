package mcp

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Ensure InMemTransport implements Transport interface
var _ Transport = (*InMemTransport)(nil)

// transportState represents the state of the transport
type transportState int32

const (
	stateStopped transportState = iota
	stateRunning
)

type InMemTransport struct {
	options TransportOptions

	// Channels initialized in Start()
	ch       chan JSONRPCMessage
	outgoing chan JSONRPCMessage
	incoming chan JSONRPCMessage
	done     chan struct{}

	// WaitGroup to track background goroutines
	wg sync.WaitGroup

	state atomic.Int32
}

func NewInMemTransport(opts ...TransportOption) *InMemTransport {
	options := TransportOptions{}
	for _, apply := range opts {
		apply(&options)
	}
	return &InMemTransport{
		options: options,
	}
}

func (t *InMemTransport) Start(ctx context.Context) error {
	if !t.state.CompareAndSwap(int32(stateStopped), int32(stateRunning)) {
		return ErrTransportStarted
	}

	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		t.state.Store(int32(stateStopped))
		return err
	}

	// Initialize channels
	t.ch = make(chan JSONRPCMessage, 1)
	t.outgoing = make(chan JSONRPCMessage, 100)
	t.incoming = make(chan JSONRPCMessage, 100)
	t.done = make(chan struct{})

	readReady := make(chan struct{})
	writeReady := make(chan struct{})

	// Start outgoing message loop
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		close(writeReady)
		t.outgoingLoop()
	}()

	// Start incoming message loop
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		close(readReady)
		t.incomingLoop()
	}()

	// Wait for loops to be "ready"
	<-readReady
	<-writeReady

	return nil
}

func (t *InMemTransport) outgoingLoop() {
	for {
		select {
		case <-t.done:
			return
		case msg := <-t.outgoing:
			// Apply send delay if configured
			if t.options.SendDelay > 0 {
				time.Sleep(t.options.SendDelay)
			}

			select {
			case <-t.done:
				return
			case t.ch <- msg:
			}
		}
	}
}

func (t *InMemTransport) incomingLoop() {
	for {
		select {
		case <-t.done:
			return
		case msg := <-t.ch:
			// Apply receive delay if configured
			if t.options.RecvDelay > 0 {
				time.Sleep(t.options.RecvDelay)
			}

			select {
			case <-t.done:
				return
			case t.incoming <- msg:
			}
		}
	}
}

func (t *InMemTransport) Send(ctx context.Context, msg JSONRPCMessage) error {
	if t.state.Load() != int32(stateRunning) {
		return ErrTransportClosed
	}

	// exit early if context is cancelled
	// NOTE this is here to hack around tests
	// as we cant guarantee the order of select branches
	if err := ctx.Err(); err != nil {
		return err
	}

	if t.options.SendTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.options.SendTimeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.outgoing <- msg:
		return nil
	}
}

func (t *InMemTransport) Receive(ctx context.Context) (JSONRPCMessage, error) {
	if t.state.Load() != int32(stateRunning) {
		return nil, ErrTransportClosed
	}

	// exit early if context is cancelled
	// NOTE this is here to hack around tests
	// as we cant guarantee the order of select branches
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if t.options.RecvTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.options.RecvTimeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-t.incoming:
		return msg, nil
	}
}

func (t *InMemTransport) Close() error {
	if !t.state.CompareAndSwap(int32(stateRunning), int32(stateStopped)) {
		return nil
	}

	// Signal loops to stop
	close(t.done)

	// Wait for loops to finish
	t.wg.Wait()

	// Close channels
	close(t.ch)
	close(t.outgoing)
	close(t.incoming)

	return nil
}
