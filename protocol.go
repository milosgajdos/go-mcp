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
	DefaultReqTimeout = 60 * time.Second
	// DefaultRespTimeout is set to 30s.
	DefaultRespTimeout = 60 * time.Second
	// RPCHandleTimeout sets max timeout to handle ia JSON RPC message.
	RPCHandleTimeout = 60 * time.Second
)

var (
	// ErrTransportClosed is returned when the transport has been closed.
	ErrTransportClosed = errors.New("transport closed")
	// ErrInvalidTransport is returned when attempting to connect using invalid transport.
	ErrInvalidTransport = errors.New("invalid transport")
	// ErrPendingRequests is returned when attempting to connect while there are unhandled requests.
	ErrPendingRequests = errors.New("pending requests")
	// ErrResponseTimeout is returned when a request times out.
	ErrResponseTimeout = errors.New("respose timed out")
	// ErrInvalidMessage is returned when an invalid message is handled.
	ErrInvalidMessage = errors.New("invalid message")
	// ErrAlreadyConnected is returned when a connected protcol is attempted to connect again.
	ErrAlreadyConnected = errors.New("already connected")
)

// Options are client options
type Options struct {
	ReqTimeout  time.Duration
	RespTimeout time.Duration
}

// Option is functional graph option.
type Option func(*Options)

func DefaultOptions() Options {
	return Options{
		ReqTimeout:  DefaultReqTimeout,
		RespTimeout: DefaultRespTimeout,
	}
}

// WithReqTimeout sets request timeout option.
func WithReqTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.ReqTimeout = timeout
	}
}

// WithRespTimeout sets request timeout option.
func WithRespTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.RespTimeout = timeout
	}
}

// RequestHandler for handling JSON RPC requests.
type RequestHandler[T ID, RQ GenRequest[T], RS GenResult] func(context.Context, *JSONRPCRequest[T, RQ]) (RS, error)

// NotificationHandler for handing JSON RPC notifications.
type NotificationHandler[T ID, NF GenNotification[T]] func(context.Context, *JSONRPCNotification[T, NF]) error

// Go doesn's provide sum types so this is our "union"
// JSONRPCResponse | JSONRPCError
type RespOrError[T ID, RS GenResult] struct {
	Resp *JSONRPCResponse[T, RS]
	Err  *JSONRPCError[T]
}

// Protocol is a Transport agnostic implementation of MCP communication protocol.
type Protocol[T ID, RQ GenRequest[T], NF GenNotification[T], RS GenResult] struct {
	transport Transport

	// For tracking pending requests
	pendingMu sync.RWMutex
	pending   map[RequestID[T]]chan RespOrError[T, RS]
	nextID    atomic.Uint64

	// request handlers
	handlersMu sync.RWMutex
	handlers   map[RequestMethod]RequestHandler[T, RQ, RS]

	// notification handlers
	notifyMu sync.RWMutex
	notify   map[RequestMethod]NotificationHandler[T, NF]

	// Error propagation
	errChan chan error

	// Lifetime management
	ctx    context.Context
	cancel context.CancelFunc

	// Track if receive loop is running
	running atomic.Bool
}

// NewProtocol creates a new instances of Protocol and returns it.
// TODO: make Transport an optional parameter and use stdio as the default transport.
func NewProtocol[
	T ID,
	RQ GenRequest[T],
	NF GenNotification[T],
	RS GenResult,
](transport Transport) *Protocol[T, RQ, NF, RS] {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Protocol[T, RQ, NF, RS]{
		transport: transport,
		pending:   make(map[RequestID[T]]chan RespOrError[T, RS]),
		handlers:  make(map[RequestMethod]RequestHandler[T, RQ, RS]),
		notify:    make(map[RequestMethod]NotificationHandler[T, NF]),
		errChan:   make(chan error, 1),
		ctx:       ctx,
		cancel:    cancel,
	}

	//Register default ping handler
	p.RegisterRequestHandler(PingRequestMethod, func(context.Context, *JSONRPCRequest[T, RQ]) (RS, error) {
		// TODO: consider adding Pong Result
		// so we don't return empty Result.
		return any(Result{}).(RS), nil
	})

	return p
}

// Connect establishes the protocol on top of the given transport.
// If protocol transport is not initialized it returns error.
// If there are pending requests it returns error.
func (p *Protocol[T, RQ, NF, RS]) Connect() error {
	if !p.running.CompareAndSwap(false, true) {
		return ErrAlreadyConnected
	}

	p.pendingMu.RLock()
	defer p.pendingMu.RUnlock()

	if len(p.pending) > 0 {
		return ErrPendingRequests
	}

	if p.transport == nil {
		return ErrInvalidTransport
	}

	if err := p.ctx.Err(); err != nil {
		// protocol has been closed
		if err == context.Canceled {
			p.ctx, p.cancel = context.WithCancel(context.Background())
		}
		return err
	}

	go p.recvMsg()

	return nil
}

// Close terminates the protocol and its transport.
func (p *Protocol[T, RQ, NF, RS]) Close(ctx context.Context) error {
	defer p.running.Store(false)
	defer p.cancel()

	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()

	if p.transport != nil {
		if err := p.transport.Close(); err != nil {
			return err
		}
	}

	// notify all pending requests
	// NOTE: if we got here, the transport has been closed
	// so sending anything down its pipes will fail.
	for reqID, ch := range p.pending {
		errResp := &JSONRPCError[T]{
			ID:      reqID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCConnectionClosed,
				Message: ErrTransportClosed.Error(),
			},
		}
		select {
		case ch <- RespOrError[T, RS]{Err: errResp}:
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
		delete(p.pending, reqID)
	}

	return nil
}

func (p *Protocol[T, RQ, NF, RS]) SendRequest(
	ctx context.Context,
	req *JSONRPCRequest[T, RQ],
	opts ...Option,
) (*JSONRPCResponse[T, RS], error) {
	options := DefaultOptions()
	for _, apply := range opts {
		apply(&options)
	}

	// Returns the old value and increments
	id := p.nextID.Add(1)

	req.ID = RequestID[T]{Value: T(id)}
	req.Version = JSONRPCVersion

	// Create response channel and register request
	respChan := make(chan RespOrError[T, RS], 1)

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
		return nil, fmt.Errorf("send request: %v", err)
	}

	// Create request timeout context
	reqCtx, cancel := context.WithTimeout(ctx, options.ReqTimeout)
	defer cancel()

	if err := p.transport.Send(reqCtx, data); err != nil {
		return nil, err
	}

	// Create request timeout context
	respCtx, cancel := context.WithTimeout(ctx, options.RespTimeout)
	defer cancel()

	// send succeeded, let's wait for the response
	select {
	case result := <-respChan:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Resp, nil
	case <-respCtx.Done():
		if respCtx.Err() == context.DeadlineExceeded {
			return nil, Error{
				Code:    JSONRPCRequestTimeout,
				Message: ErrResponseTimeout.Error(),
			}
		}
		return nil, respCtx.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-p.errChan:
		return nil, fmt.Errorf("protocol error: %w", err)
	}
}

// SendNotification sends a notification (fire and forget)
func (p *Protocol[T, RQ, NF, RS]) SendNotification(ctx context.Context, notif *JSONRPCNotification[T, NF]) error {
	notif.Version = JSONRPCVersion // making sure we have the right version
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	return p.transport.Send(ctx, data)
}

// RegisterRequestHandler registers a handler for the given request method.
// If there is an existing handler registered, this method overrides it.
func (p *Protocol[T, RQ, NF, RS]) RegisterRequestHandler(method RequestMethod, handler RequestHandler[T, RQ, RS]) {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	p.handlers[method] = handler
}

// RegisterNotificationHandler registers a handler for the given notification request method.
// If there is an existing handler registered, this method overrides it.
func (p *Protocol[T, RQ, NF, RS]) RegisterNotificationHandler(method RequestMethod, handler NotificationHandler[T, NF]) {
	p.notifyMu.Lock()
	defer p.notifyMu.Unlock()
	p.notify[method] = handler
}

// recvMsg handles incoming messages from the transport
func (p *Protocol[T, RQ, NF, RS]) recvMsg() {
	defer p.running.Store(false)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			// TODO: this might be blocking on errChan
			// we should consider some timeout here and then
			// maybe log the error if nothing collects it.
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
func (p *Protocol[T, RQ, NF, RS]) sendResponse(ctx context.Context, resp any) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	if err := p.transport.Send(ctx, data); err != nil {
		return fmt.Errorf("send response: %v", err)
	}

	return nil
}

func (p *Protocol[T, RQ, NF, RS]) handleError(ctx context.Context, err *JSONRPCError[T]) {
	p.pendingMu.RLock()
	ch, ok := p.pending[err.ID]
	p.pendingMu.RUnlock()

	if ok {
		select {
		case ch <- RespOrError[T, RS]{Err: err}:
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
	}
}

func (p *Protocol[T, RQ, NF, RS]) handleResponse(ctx context.Context, resp *JSONRPCResponse[T, RS]) {
	p.pendingMu.RLock()
	ch, ok := p.pending[resp.ID]
	p.pendingMu.RUnlock()

	if ok {
		select {
		case ch <- RespOrError[T, RS]{Resp: resp}:
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
	}
}

func (p *Protocol[T, RQ, NF, RS]) handleRequest(ctx context.Context, req *JSONRPCRequest[T, RQ]) {
	p.handlersMu.RLock()
	handler, ok := p.handlers[req.Request.GetMethod()]
	p.handlersMu.RUnlock()

	// No handler exists for this method
	// We must send JSONRPCMethodNotFoundError
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
			case p.errChan <- err:
			case <-p.ctx.Done():
			case <-ctx.Done():
			}
		}
		return
	}

	result, err := handler(ctx, req)
	if err != nil {
		errResp := &JSONRPCError[T]{
			ID:      req.ID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCInternalError,
				Message: err.Error(),
			},
		}
		// If it's a context error, let's create a nre context.
		if ctx.Err() != nil {
			ctx = context.Background()
		}
		if err := p.sendResponse(ctx, errResp); err != nil {
			errResp.Err.Message = fmt.Sprintf("%s: %v", errResp.Err, err)
			select {
			case p.errChan <- errResp:
			case <-p.ctx.Done():
			case <-ctx.Done():
			}
		}
		return
	}

	// Send success response
	resp := &JSONRPCResponse[T, RS]{
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
		case p.errChan <- errResp:
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
	}
}

func (p *Protocol[T, RQ, NF, RS]) handleNotification(ctx context.Context, n *JSONRPCNotification[T, NF]) {
	p.notifyMu.RLock()
	handler, ok := p.notify[n.Notification.GetMethod()]
	p.notifyMu.RUnlock()

	if ok {
		// We can ignore errors from notification handlers
		// since notifications don't expect responses
		// We could log the error here or something
		_ = handler(ctx, n)
	}
}

// handleMessage processes a received JSON RPC message.
// NOTE: msg:  JSONRPCRequest | JSONRPCNotification | JSONRPCResponse | JSONRPCError
func (p *Protocol[T, RQ, NF, RS]) handleMessage(msg []byte) {
	// Create request-specific context
	ctx, cancel := context.WithTimeout(context.Background(), RPCHandleTimeout)
	defer cancel()

	var errResp JSONRPCError[T]
	if err := json.Unmarshal(msg, &errResp); err == nil {
		p.handleError(ctx, &errResp)
		return
	}

	var resp JSONRPCResponse[T, RS]
	if err := json.Unmarshal(msg, &resp); err == nil {
		p.handleResponse(ctx, &resp)
		return
	}

	var req JSONRPCRequest[T, RQ]
	if err := json.Unmarshal(msg, &req); err == nil {
		p.handleRequest(ctx, &req)
		return
	}

	// Finally try as notification
	var notif JSONRPCNotification[T, NF]
	if err := json.Unmarshal(msg, &notif); err == nil {
		p.handleNotification(ctx, &notif)
		return
	}

	// If we get here, the message wasn't valid JSON-RPC
	// TODO: consider adding some timeout here as we might be
	// blocking here on errChan send
	select {
	case p.errChan <- ErrInvalidMessage:
	case <-p.ctx.Done():
	}
}
