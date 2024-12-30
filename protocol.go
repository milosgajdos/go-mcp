package mcp

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultReqTimeout sets max timeout for receiving a JSON RPC response.
	DefaultReqTimeout = 60 * time.Second
	// DefaultHandleTimeout sets max timeout for handling a JSON RPC message.
	DefaultHandleTimeout = 60 * time.Second
	// DefaultWaitTimeout sets max wait timeout for a JSON RPC message to be consumed.
	DefaultWaitTimeout = 300 * time.Millisecond
)

var (
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
	ReqTimeout    time.Duration
	HandleTimeout time.Duration
	WaitTimeout   time.Duration
	Transport     Transport
}

// Option is functional graph option.
type Option func(*Options)

func DefaultOptions() Options {
	return Options{
		ReqTimeout:    DefaultReqTimeout,
		HandleTimeout: DefaultHandleTimeout,
		WaitTimeout:   DefaultWaitTimeout,
		Transport:     NewInMemTransport(),
	}
}

// WithReqTimeout sets request timeout option.
func WithReqTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.ReqTimeout = timeout
	}
}

// WithHandleTimeout sets message handling timeout option.
func WithHandleTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.HandleTimeout = timeout
	}
}

// WithWaitTimeout sets response stream timeout option.
func WithWaitTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.WaitTimeout = timeout
	}
}

// WithTransport sets protocol transport.
func WithTransport(tr Transport) Option {
	return func(o *Options) {
		o.Transport = tr
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

	// protocol options
	options Options

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
func NewProtocol[T ID](opts ...Option) *Protocol[T] {
	options := DefaultOptions()
	for _, apply := range opts {
		apply(&options)
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Protocol[T]{
		transport: options.Transport,
		options:   options,
		pending:   make(map[RequestID[T]]chan RespOrError[T]),
		handlers:  make(map[RequestMethod]RequestHandler[T]),
		notify:    make(map[RequestMethod]NotificationHandler[T]),
		ctx:       ctx,
		cancel:    cancel,
	}

	//Register default ping handler
	p.RegisterRequestHandler(PingRequestMethod,
		func(_ context.Context, req *JSONRPCRequest[T]) (*JSONRPCResponse[T], error) {
			return &JSONRPCResponse[T]{
				ID:      req.ID,
				Version: req.Version,
				Result:  &PingResult{},
			}, nil
		})

	return p
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

// Connect establishes the protocol on top of the given transport
// and starts receiving messages sent to it.
// TODO: consider overriding Transport on connect.
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
	for reqID, ch := range p.pending {
		errResp := &JSONRPCError[T]{
			ID:      reqID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCConnectionClosed,
				Message: "Connection closed",
			},
		}
		select {
		case ch <- RespOrError[T]{Err: errResp}:
		case <-time.After(p.options.WaitTimeout):
			log.Printf("response send to %v request timed out", reqID)
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
		delete(p.pending, reqID)
	}

	return nil
}

// SendRequest sends a request and waits for response
// TODO: consider overriding protocol timeouts.
func (p *Protocol[T]) SendRequest(ctx context.Context, req *JSONRPCRequest[T]) (*JSONRPCResponse[T], error) {
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

	if err := p.transport.Send(ctx, req); err != nil {
		return nil, err
	}

	select {
	case result := <-respChan:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Resp, nil
	case <-time.After(p.options.ReqTimeout):
		return nil, &JSONRPCError[T]{
			ID: req.ID,
			Err: Error{
				Code:    JSONRPCRequestTimeout,
				Message: ErrResponseTimeout.Error(),
			},
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

// SendNotification sends a notification (fire and forget)
func (p *Protocol[T]) SendNotification(ctx context.Context, notif *JSONRPCNotification[T]) error {
	notif.Version = JSONRPCVersion
	return p.transport.Send(ctx, notif)
}

// recvMsg handles incoming messages from the transport
func (p *Protocol[T]) recvMsg() {
	defer p.running.Store(false)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			// TODO: consider recovering from possible
			// intermittent Transport errors
			msg, err := p.transport.Receive(p.ctx)
			if err != nil {
				if closeErr := p.Close(context.Background()); closeErr != nil {
					log.Printf("shutting down: %v", err)
				}
				return
			}
			go p.handleMessage(msg)
		}
	}
}

func (p *Protocol[T]) handleMessage(msg JSONRPCMessage) {
	// NOTE: this should never happen if underlying Transport
	// is properly implemented, but let's not take that risk!
	if msg == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.options.HandleTimeout)
	defer cancel()

	switch msg.JSONRPCMessageType() {
	case JSONRPCRequestMsgType:
		req, ok := msg.(*JSONRPCRequest[T])
		if !ok {
			log.Printf("invalid request message type: %T", msg)
			return
		}
		p.handleRequest(ctx, req)

	case JSONRPCNotificationMsgType:
		notif, ok := msg.(*JSONRPCNotification[T])
		if !ok {
			log.Printf("invalid notification message type: %T", msg)
			return
		}
		p.handleNotification(ctx, notif)

	case JSONRPCResponseMsgType:
		resp, ok := msg.(*JSONRPCResponse[T])
		if !ok {
			log.Printf("invalid response message type: %T", msg)
			return
		}
		p.handleResponse(ctx, resp)

	case JSONRPCErrorMsgType:
		errMsg, ok := msg.(*JSONRPCError[T])
		if !ok {
			log.Printf("invalid error message type: %T", msg)
			return
		}
		p.handleError(ctx, errMsg)
	}
}

func (p *Protocol[T]) handleError(ctx context.Context, errMsg *JSONRPCError[T]) {
	p.pendingMu.RLock()
	ch, ok := p.pending[errMsg.ID]
	p.pendingMu.RUnlock()

	if ok {
		select {
		case ch <- RespOrError[T]{Err: errMsg}:
		case <-time.After(p.options.WaitTimeout):
			log.Printf("handle RPC handle error client wait timeout")
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
	}
}

func (p *Protocol[T]) handleResponse(ctx context.Context, resp *JSONRPCResponse[T]) {
	p.pendingMu.RLock()
	ch, ok := p.pending[resp.ID]
	p.pendingMu.RUnlock()

	if ok {
		select {
		case ch <- RespOrError[T]{Resp: resp}:
		case <-time.After(p.options.WaitTimeout):
			log.Printf("handle RPC handle response client wait timeout")
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
	}
}

func (p *Protocol[T]) handleNotification(ctx context.Context, n *JSONRPCNotification[T]) {
	p.notifyMu.RLock()
	handler, ok := p.notify[n.Notification.GetMethod()]
	p.notifyMu.RUnlock()

	if ok {
		if err := handler(ctx, n); err != nil {
			log.Printf("notification handler error: %v", err)
		}
	}
}

func (p *Protocol[T]) handleRequest(ctx context.Context, req *JSONRPCRequest[T]) {
	// NOTE: this should never happen if underlying Transport
	// is properly implemented, but let's not take that risk!
	if req.Request == nil {
		log.Printf("Received empty JSON-RPC request with: %#v", req)
		return
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
		if err := p.transport.Send(ctx, errResp); err != nil {
			log.Printf("send request error response: %v", err)
		}
		return
	}

	resp, err := handler(ctx, req)
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
		if err := p.transport.Send(ctx, errResp); err != nil {
			log.Printf("send handler error response: %v", err)
		}
		return
	}

	out := &JSONRPCResponse[T]{
		ID:      req.ID,
		Version: JSONRPCVersion,
		Result:  resp.Result,
	}
	if err := p.transport.Send(ctx, out); err != nil {
		log.Printf("send handler request response: %v", err)
	}
}
