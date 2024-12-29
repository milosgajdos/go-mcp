package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultReqTimeout sets max timeout for sending a JSON RPC request.
	DefaultReqTimeout = 60 * time.Second
	// DefaultRespTimeout sets max timeout for receiving a JSON RPC response.
	DefaultRespTimeout = 60 * time.Second
	// HandleTimeout sets max timeout for handling a JSON RPC message.
	HandleTimeout = 60 * time.Second
	// WaitTimeout sets max wait timeout for a JSON RPC message to be consumed.
	WaitTimeout = 300 * time.Millisecond
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
type RequestHandler[T ID] func(context.Context, *JSONRPCRequest[T]) (*JSONRPCResponse[T], error)

// NotificationHandler for handing JSON RPC notifications.
type NotificationHandler[T ID] func(context.Context, *JSONRPCNotification[T]) error

// Go doesn's provide sum types so this is our "union"
// JSONRPCResponse | JSONRPCError
type RespOrError[T ID] struct {
	Resp *JSONRPCResponse[T]
	Err  *JSONRPCError[T]
}

// Protocol is a Transport agnostic implementation of MCP communication protocol.
type Protocol[T ID] struct {
	transport Transport

	// For tracking pending requests
	pendingMu sync.RWMutex
	pending   map[RequestID[T]]chan RespOrError[T]
	nextID    atomic.Uint64

	// request handlers
	handlersMu sync.RWMutex
	handlers   map[RequestMethod]RequestHandler[T]

	// notification handlers
	notifyMu sync.RWMutex
	notify   map[RequestMethod]NotificationHandler[T]

	// Lifetime management
	ctx    context.Context
	cancel context.CancelFunc

	// Track if receive loop is running
	running atomic.Bool
}

// NewProtocol creates a new instances of Protocol and returns it.
// TODO: make Transport an optional parameter and use stdio as the default transport.
func NewProtocol[T ID](transport Transport) *Protocol[T] {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Protocol[T]{
		transport: transport,
		pending:   make(map[RequestID[T]]chan RespOrError[T]),
		handlers:  make(map[RequestMethod]RequestHandler[T]),
		notify:    make(map[RequestMethod]NotificationHandler[T]),
		ctx:       ctx,
		cancel:    cancel,
	}

	//Register default ping handler
	p.RegisterRequestHandler(PingRequestMethod, func(_ context.Context, req *JSONRPCRequest[T]) (*JSONRPCResponse[T], error) {
		// TODO: consider adding Pong Result
		return &JSONRPCResponse[T]{
			ID:      req.ID,
			Version: req.Version,
		}, nil
	})

	return p
}

// Connect establishes the protocol on top of the given transport.
// If protocol transport is not initialized it returns error.
// If there are pending requests it returns error.
func (p *Protocol[T]) Connect() error {
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
		switch err {
		case context.Canceled: // protocol has been closed
			p.ctx, p.cancel = context.WithCancel(context.Background())
		default: // some other context error
			return err
		}
	}

	go p.recvMsg()

	return nil
}

// Close terminates the protocol and its transport.
func (p *Protocol[T]) Close(ctx context.Context) error {
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
		case ch <- RespOrError[T]{Err: errResp}:
		case <-p.ctx.Done():
		case <-ctx.Done():
		case <-time.After(WaitTimeout):
			log.Printf("response send to %v request timed out", reqID)
		}
		delete(p.pending, reqID)
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
	respChan := make(chan RespOrError[T], 1)

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
	}
}

// SendNotification sends a notification (fire and forget)
func (p *Protocol[T]) SendNotification(ctx context.Context, notif *JSONRPCNotification[T]) error {
	notif.Version = JSONRPCVersion // making sure we have the right version
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
	defer p.handlersMu.Unlock()
	p.handlers[method] = handler
}

// DeregisterRequestHandler deregisters a handler for the given request method.
func (p *Protocol[T]) DeregisterRequestHandler(method RequestMethod) {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	delete(p.handlers, method)
}

// RegisterNotificationHandler registers a handler for the given notification request method.
// If there is an existing handler registered, this method overrides it.
func (p *Protocol[T]) RegisterNotificationHandler(method RequestMethod, handler NotificationHandler[T]) {
	p.notifyMu.Lock()
	defer p.notifyMu.Unlock()
	p.notify[method] = handler
}

// DeregisterNotificationHandler deregisters a handler for the given request method.
func (p *Protocol[T]) DeregisterNotificationHandler(method RequestMethod) {
	p.notifyMu.Lock()
	defer p.notifyMu.Unlock()
	delete(p.notify, method)
}

// recvMsg handles incoming messages from the transport
func (p *Protocol[T]) recvMsg() {
	defer p.running.Store(false)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			// TODO: consider transport recovery
			// I/O errors may be transient so we
			// should do some best effort retry.
			msg, err := p.transport.Receive(p.ctx)
			if err != nil {
				log.Printf("receive message: %v", err)
				if closeErr := p.Close(context.Background()); closeErr != nil {
					log.Printf("shutting down: %v", err)
				}
				return
			}
			// TODO: we need to consider some bounded concurrency here
			// so that under high load we limit the number of goroutines.
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
		return fmt.Errorf("send response: %v", err)
	}

	return nil
}

func (p *Protocol[T]) handleError(ctx context.Context, id RequestID[T], msg []byte) error {
	p.pendingMu.RLock()
	ch, ok := p.pending[id]
	p.pendingMu.RUnlock()

	if ok {
		var errResp JSONRPCError[T]
		if err := json.Unmarshal(msg, &errResp); err != nil {
			errResp.ID = id
			errResp.Version = JSONRPCVersion
			errResp.Err.Code = JSONRPCParseError
			errResp.Err.Message = fmt.Sprintf("invalid RPC error message: %v", err)
		}

		select {
		case ch <- RespOrError[T]{Err: &errResp}:
			return nil
		case <-time.After(WaitTimeout):
			return fmt.Errorf("handle RPC error timeout")
		case <-p.ctx.Done():
			return p.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (p *Protocol[T]) handleResponse(ctx context.Context, id RequestID[T], msg []byte) error {
	p.pendingMu.RLock()
	ch, ok := p.pending[id]
	p.pendingMu.RUnlock()

	if ok {
		var respOrErr RespOrError[T]

		var resp JSONRPCResponse[T]
		if err := json.Unmarshal(msg, &resp); err != nil {
			respOrErr.Err = &JSONRPCError[T]{
				ID:      id,
				Version: JSONRPCVersion,
				Err: Error{
					Code:    JSONRPCParseError,
					Message: fmt.Sprintf("invalid RPC response message: %v", err),
				},
			}
		}
		if respOrErr.Err == nil {
			respOrErr.Resp = &resp
		}

		select {
		case ch <- respOrErr:
			return nil
		case <-time.After(WaitTimeout):
			return fmt.Errorf("handle RPC response timeout")
		case <-p.ctx.Done():
			return p.ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (p *Protocol[T]) handleRequest(ctx context.Context, id RequestID[T], msg []byte) error {
	var req JSONRPCRequest[T]
	if err := json.Unmarshal(msg, &req); err != nil {
		errResp := &JSONRPCError[T]{
			ID:      id,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCParseError,
				Message: fmt.Sprintf("invalid RPC request message: %v", err),
			},
		}
		if err := p.sendResponse(ctx, errResp); err != nil {
			return fmt.Errorf("send response: %v", errResp)
		}
		return fmt.Errorf("invalid RPC request: %v", err)
	}

	p.handlersMu.RLock()
	handler, ok := p.handlers[req.Request.GetMethod()]
	p.handlersMu.RUnlock()

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
			return fmt.Errorf("send response: %v", err)
		}
		return nil
	}

	resp, err := handler(ctx, &req)
	if err != nil {
		errResp := &JSONRPCError[T]{
			ID:      req.ID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCInternalError,
				Message: err.Error(),
			},
		}
		// If it's a context error, let's create a new context.
		if ctx.Err() != nil {
			ctx = context.Background()
		}
		if err := p.sendResponse(ctx, errResp); err != nil {
			return fmt.Errorf("send response: %v", err)
		}
		return nil
	}

	// Send success response
	out := &JSONRPCResponse[T]{
		ID:      req.ID,
		Version: JSONRPCVersion,
		Result:  resp.Result,
	}
	if err := p.sendResponse(ctx, out); err != nil {
		return fmt.Errorf("send response: %v", err)
	}
	return nil
}

func (p *Protocol[T]) handleNotification(ctx context.Context, msg []byte) error {
	var n JSONRPCNotification[T]
	if err := json.Unmarshal(msg, &n); err != nil {
		return fmt.Errorf("invalid RPC notification: %v", err)
	}

	p.notifyMu.RLock()
	handler, ok := p.notify[n.Notification.GetMethod()]
	p.notifyMu.RUnlock()

	if ok {
		// We can ignore errors from notification handlers
		// since notifications don't expect responses
		// We could log the error here or something
		_ = handler(ctx, &n)
	}
	return nil
}

// handleMessage processes a received JSON RPC message.
func (p *Protocol[T]) handleMessage(msg []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), HandleTimeout)
	defer cancel()

	type Base struct {
		Version string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  json.RawMessage `json:"method"`
		Error   json.RawMessage `json:"error"`
	}

	var rpcMsg Base
	if err := json.Unmarshal(msg, &rpcMsg); err != nil {
		log.Printf("invalid RPC message: %v", err)
		return
	}

	if len(rpcMsg.ID) > 0 {
		var msgID RequestID[T]
		if err := json.Unmarshal(rpcMsg.ID, &msgID); err != nil {
			log.Printf("failed to unmarshal message 'id': %v", err)
			return
		}

		if len(rpcMsg.Error) > 0 {
			if err := p.handleError(ctx, msgID, msg); err != nil {
				log.Printf("handle RPC error: %v", err)
			}
			return
		}

		if len(rpcMsg.Method) == 0 {
			if err := p.handleResponse(ctx, msgID, msg); err != nil {
				log.Printf("handle RPC response: %v", err)
			}
			return
		}

		if err := p.handleRequest(ctx, msgID, msg); err != nil {
			log.Printf("handle RPC request: %v", err)
		}
		return
	}

	if err := p.handleNotification(ctx, msg); err != nil {
		log.Printf("handle RPC notification: %v", err)
		return
	}
}
