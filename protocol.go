package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultReqTimeout is set to 30s.
	DefaultReqTimeout = 30 * time.Second
)

// Options are client options
type Options struct {
	ReqTimeout time.Duration
}

// Option is functional graph option.
type Option func(*Options)

func DefaultOptions() Options {
	return Options{
		ReqTimeout: DefaultReqTimeout,
	}
}

// WithTimeout sets request timeout option
func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.ReqTimeout = timeout
	}
}

var (
	// ErrTransportClosed is returned when the transport has been closed.
	ErrTransportClosed = errors.New("transport closed")
	// ErrInvalidTransport is returned when attempting to connect using invalid transport.
	ErrInvalidTransport = errors.New("invalid transport")
	// ErrPendingRequests is returned when attempting to connect while there are unhandled requests.
	ErrPendingRequests = errors.New("pending requests")
	// ErrRequestTimeout is returned when a request times out.
	ErrRequestTimeout = errors.New("request timed out")
	// ErrInvalidMessage is returned when an invalid message is handled.
	ErrInvalidMessage = errors.New("invalid message")
	// ErrAlreadyConnected is returned when a connected protcol is attempted to connect again.
	ErrAlreadyConnected = errors.New("already connected")
)

// RequestHandler for handling RPC requests.
type RequestHandler[T ID] func(context.Context, *JSONRPCRequest[T]) (Result, error)

// NotificationHandler for handing RPC notifications.
type NotificationHandler func(context.Context, *JSONRPCNotification) error

// ResponseOrError tracks responses for specific requests
type ResponseOrError[T ID] struct {
	Response *JSONRPCResponse[T]
	Error    *JSONRPCError[T]
}

// Protocol is a Transport agnostic implementation of MCP communication protocol.
type Protocol[T ID] struct {
	transport Transport

	// For tracking pending requests
	pendingMu sync.RWMutex
	pending   map[RequestID[T]]chan ResponseOrError[T]
	nextID    atomic.Uint64

	// For request handlers
	handlersMu sync.RWMutex
	handlers   map[RequestMethod]RequestHandler[T]

	// For notification handlers
	notifyMu sync.RWMutex
	notify   map[RequestMethod]NotificationHandler

	// Error propagation
	errChan chan error

	// Lifetime management
	ctx    context.Context
	cancel context.CancelFunc

	// Track if receive loop is running
	running atomic.Bool
}

// NewProtocol creates a new instances of Protocol and returns it.
func NewProtocol[T ID](transport Transport) *Protocol[T] {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Protocol[T]{
		transport: transport,
		pending:   make(map[RequestID[T]]chan ResponseOrError[T]),
		handlers:  make(map[RequestMethod]RequestHandler[T]),
		notify:    make(map[RequestMethod]NotificationHandler),
		errChan:   make(chan error, 1),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Register default ping handler
	p.RegisterRequestHandler(PingRequestMethod, func(_ context.Context, _ *JSONRPCRequest[T]) (Result, error) {
		return Result{}, nil
	})

	return p
}

// Connect establishes the protocol on top of the given transport.
// If protocol transport is not initialized it returns error.
// If there are pending requests it returns error.
func (p *Protocol[T]) Connect() error {
	p.pendingMu.RLock()
	defer p.pendingMu.RUnlock()

	if len(p.pending) > 0 {
		return ErrPendingRequests
	}

	if p.transport == nil {
		return ErrInvalidTransport
	}

	// Only start if not already running
	if !p.running.CompareAndSwap(false, true) {
		return ErrAlreadyConnected
	}

	go p.receive()

	return nil
}

// Close terminates the protocol and its transport.
func (p *Protocol[T]) Close() error {
	p.cancel()

	p.pendingMu.RLock()
	defer p.pendingMu.RUnlock()

	if p.transport != nil {
		return p.transport.Close()
	}
	return nil
}

func (p *Protocol[T]) SendRequest(ctx context.Context, req *JSONRPCRequest[T], opts ...Option) (*JSONRPCResponse[T], error) {
	options := DefaultOptions()
	for _, apply := range opts {
		apply(&options)
	}

	// Returns the old value and increments
	id := p.nextID.Add(1)

	req.ID = RequestID[T]{Value: T(id)}
	req.Version = JSONRPCVersion

	// Create response channel and register request
	respChan := make(chan ResponseOrError[T], 1)

	p.pendingMu.Lock()
	p.pending[req.ID] = respChan
	p.pendingMu.Unlock()

	// Clean up on exit
	defer func() {
		p.pendingMu.Lock()
		delete(p.pending, req.ID)
		p.pendingMu.Unlock()
	}()

	// Send the request using the original context
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Send request asynchronously
	sendDone := make(chan error, 1)
	go func() {
		sendDone <- p.transport.Send(ctx, data)
	}()

	// Wait for send completion
	if err := <-sendDone; err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, options.ReqTimeout)
	defer cancel()

	// send succeeded, let's wait for the response
	select {
	case result := <-respChan:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Response, nil
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, Error{
				Code:    JSONRPCRequestTimeout,
				Message: ErrRequestTimeout.Error(),
			}
		}
		return nil, timeoutCtx.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-p.errChan:
		return nil, fmt.Errorf("protocol error: %w", err)
	}
}

// SendNotification sends a notification (fire and forget)
func (p *Protocol[T]) SendNotification(ctx context.Context, notif *JSONRPCNotification) error {
	// make sure we have the right version
	notif.Version = JSONRPCVersion
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	return p.transport.Send(ctx, data)
}

// RegisterRequestHandler registers a handler for the given request method.
// If there is an existing handler registered, this method overrides it.
func (p *Protocol[T]) RegisterRequestHandler(method RequestMethod, handler RequestHandler[T]) {
	p.handlersMu.Lock()
	p.handlers[method] = handler
	p.handlersMu.Unlock()
}

// RegisterNotificationHandler registers a handler for the given notification request method.
// If there is an existing handler registered, this method overrides it.
func (p *Protocol[T]) RegisterNotificationHandler(method RequestMethod, handler NotificationHandler) {
	p.notifyMu.Lock()
	p.notify[method] = handler
	p.notifyMu.Unlock()
}

// receive handles incoming messages from the transport
func (p *Protocol[T]) receive() {
	// reset the connected bit
	defer p.running.Store(false)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			msg, err := p.transport.Receive(p.ctx)
			if err != nil {
				select {
				case p.errChan <- fmt.Errorf("receive message: %w", err):
					continue
				case <-p.ctx.Done():
					return
				}
			}
			go p.handleMessage(msg)
		}
	}
}

// Factoring out response sending to handle errors properly
// TODO: get rid of the opaque any type
func (p *Protocol[T]) sendResponse(ctx context.Context, resp any) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	if err := p.transport.Send(ctx, data); err != nil {
		return fmt.Errorf("transport send response: %w", err)
	}

	return nil
}

func (p *Protocol[T]) handleError(_ context.Context, err JSONRPCError[T]) {
	p.pendingMu.RLock()
	ch, ok := p.pending[err.ID]
	p.pendingMu.RUnlock()

	if ok {
		select {
		case ch <- ResponseOrError[T]{Error: &err}:
		case <-p.ctx.Done():
		}
	}
}

func (p *Protocol[T]) handleResponse(_ context.Context, resp JSONRPCResponse[T]) {
	p.pendingMu.RLock()
	ch, ok := p.pending[resp.ID]
	p.pendingMu.RUnlock()

	if ok {
		select {
		case ch <- ResponseOrError[T]{Response: &resp}:
		case <-p.ctx.Done():
		}
	}
}

func (p *Protocol[T]) handleRequest(ctx context.Context, req JSONRPCRequest[T]) {
	p.pendingMu.RLock()
	ch, ok := p.pending[req.ID]
	p.pendingMu.RUnlock()
	// request already handled or sender exited
	if !ok {
		return
	}

	p.handlersMu.RLock()
	handler, ok := p.handlers[req.Method]
	p.handlersMu.RUnlock()

	// No handler exists for this method
	// We must return JSONRPCMethodNotFoundError
	if !ok {
		errResp := &JSONRPCError[T]{
			ID:      req.ID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCMethodNotFoundError,
				Message: "Method not found",
			},
		}
		if err := p.sendResponse(ctx, errResp); err != nil {
			errResp.Err.Code = JSONRPCInternalError
			errResp.Err.Message = err.Error()
			select {
			case ch <- ResponseOrError[T]{
				Error: errResp,
			}:
			case <-p.ctx.Done():
			}
		}
		return
	}

	result, err := handler(ctx, &req)
	if err != nil {
		errResp := &JSONRPCError[T]{
			ID:      req.ID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCInternalError,
				Message: err.Error(),
			},
		}
		if ctx.Err() != nil {
			ctx = context.Background()
		}
		if err := p.sendResponse(ctx, errResp); err != nil {
			errResp.Err.Code = JSONRPCInternalError
			errResp.Err.Message = err.Error()
			select {
			case ch <- ResponseOrError[T]{
				Error: errResp,
			}:
			case <-p.ctx.Done():
			}
		}
		return
	}

	// Send success response
	resp := &JSONRPCResponse[T]{
		ID:      req.ID,
		Version: JSONRPCVersion,
		Result:  result,
	}
	if err := p.sendResponse(ctx, resp); err != nil {
		errResp := &JSONRPCError[T]{
			ID:      req.ID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCInternalError,
				Message: err.Error(),
			},
		}
		select {
		case ch <- ResponseOrError[T]{
			Error: errResp,
		}:
		case <-p.ctx.Done():
		}
	}
}

func (p *Protocol[T]) handleNotification(ctx context.Context, notif JSONRPCNotification) {
	p.notifyMu.RLock()
	handler, ok := p.notify[notif.Method]
	p.notifyMu.RUnlock()

	if ok {
		// We can ignore errors from notification handlers
		// since notifications don't expect responses
		// We could log the error here or something
		_ = handler(ctx, &notif)
	}
}

// handleMessage processes a received message
// NOTE: msg:  JSONRPCRequest | JSONRPCNotification | JSONRPCResponse | JSONRPCError
func (p *Protocol[T]) handleMessage(msg []byte) {
	// Create request-specific context
	// TODO: consider creating this in handleMessage()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var errResp JSONRPCError[T]
	if err := json.Unmarshal(msg, &errResp); err == nil {
		p.handleError(ctx, errResp)
		return
	}

	var resp JSONRPCResponse[T]
	if err := json.Unmarshal(msg, &resp); err == nil {
		p.handleResponse(ctx, resp)
		return
	}

	var req JSONRPCRequest[T]
	if err := json.Unmarshal(msg, &req); err == nil {
		p.handleRequest(ctx, req)
		return
	}

	// Finally try as notification
	var notif JSONRPCNotification
	if err := json.Unmarshal(msg, &notif); err == nil {
		p.handleNotification(ctx, notif)
		return
	}

	// If we get here, the message wasn't valid JSON-RPC at all
	select {
	case p.errChan <- ErrInvalidMessage:
	case <-p.ctx.Done():
	}
}
