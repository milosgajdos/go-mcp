package mcp

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrTransportStarted is returned when starting already started transport.
	ErrTransportStarted = errors.New("transport started")
	// ErrTransportClosed is returned when the transport has been closed.
	ErrTransportClosed = errors.New("transport closed")
	// ErrSendTimeout is returned when transport send times out
	ErrSendTimeout = errors.New("send timeout")
	//ErrRecvTimeout is returned when transporti receive times out
	ErrRecvTimeout = errors.New("receive timeout")
)

type TransportOptions struct {
	SendTimeout time.Duration
	RecvTimeout time.Duration
	SendDelay   time.Duration
	RecvDelay   time.Duration
}

// Option is functional graph option.
type TransportOption func(*TransportOptions)

// WithHandleTimeout sets message handling timeout option.
func WithSendTimeout(timeout time.Duration) TransportOption {
	return func(o *TransportOptions) {
		o.SendTimeout = timeout
	}
}

// WithWaitTimeout sets response stream timeout option.
func WithRecvTimeout(timeout time.Duration) TransportOption {
	return func(o *TransportOptions) {
		o.RecvTimeout = timeout
	}
}

// WithSendDelay simulates network latency by delaying send.
func WithSendDelay(delay time.Duration) TransportOption {
	return func(o *TransportOptions) {
		o.SendDelay = delay
	}
}

// WithRecvDelay simulates network latency by delaying send.
func WithRecvDelay(delay time.Duration) TransportOption {
	return func(o *TransportOptions) {
		o.RecvDelay = delay
	}
}

// Transport handles sending and receiving JSONRPC messages
type Transport interface {
	// Start starts the transport.
	Start(ctx context.Context) error
	// Send sends JSONRPCMessage to the remote endpoint.
	Send(ctx context.Context, msg JSONRPCMessage) error
	// Receive receives JSONRPCMessage from the remote endpoint.
	Receive(ctx context.Context) (JSONRPCMessage, error)
	// Close terminates the transport.
	Close() error
}
