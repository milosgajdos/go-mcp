package mcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DefaultRespTimeout sets max timeout for receiving a JSON-RPC response.
	DefaultRespTimeout = 60 * time.Second
	// DefaultHandleTimeout sets max timeout for handling a JSON-RPC message.
	DefaultHandleTimeout = 60 * time.Second
	// DefaultWaitTimeout sets max wait timeout for a JSON-RPC message to be consumed.
	DefaultWaitTimeout = 300 * time.Millisecond
)

var (
	// ErrInvalidTransport is returned when attempting to connect using invalid transport.
	ErrInvalidTransport = errors.New("invalid transport")
	// ErrPendingRequests is returned when connecting while there are pending requests.
	ErrPendingRequests = errors.New("pending requests")
	// ErrResponseTimeout is returned when a request times out.
	ErrResponseTimeout = errors.New("response timed out")
	// ErrWaitTimeout is returned when a response send times out.
	ErrWaitTimeout = errors.New("wait timed out")
	// ErrAlreadyConnected is returned when a connected protcol is attempted to connect again.
	ErrAlreadyConnected = errors.New("already connected")
)

// Options are client options
type Options struct {
	RespTimeout   time.Duration
	HandleTimeout time.Duration
	WaitTimeout   time.Duration
	Transport     Transport
	Progress      ProgressHandler
}

// Option is functional graph option.
type Option func(*Options)

func DefaultOptions() Options {
	return Options{
		RespTimeout:   DefaultRespTimeout,
		HandleTimeout: DefaultHandleTimeout,
		WaitTimeout:   DefaultWaitTimeout,
		Transport:     NewInMemTransport(),
	}
}

// WithRespTimeout sets response timeout option.
func WithRespTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.RespTimeout = timeout
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

// RequestHandler for handling JSON-RPC requests.
type RequestHandler func(ctx context.Context, n *JSONRPCRequest) (*JSONRPCResponse, error)

// NotificationHandler for handling JSON-RPC notifications.
type NotificationHandler func(ctx context.Context, n *JSONRPCNotification) error

// ProgressHandler for handling progress notifications.
type ProgressHandler func(ctx context.Context, progress float64, total *int64) error

// Go doesn's provide sum types so this is our "union"
// JSONRPCResponse | JSONRPCError
type RespOrError struct {
	Resp *JSONRPCResponse
	Err  *JSONRPCError
}

// Protocol is a Transport agnostic implementation of MCP communication protocol.
type Protocol struct {
	transport Transport

	// protocol options
	options Options

	// For tracking pending requests
	pendingMu sync.RWMutex
	pending   map[RequestID]chan RespOrError
	nextID    atomic.Uint64
	// progress notification handlers
	progress map[ProgressToken]ProgressHandler

	// request handlers
	handlersMu sync.RWMutex
	handlers   map[RequestMethod]RequestHandler

	// notification handlers
	notifyMu sync.RWMutex
	notify   map[RequestMethod]NotificationHandler

	// Lifetime management
	ctx    context.Context
	cancel context.CancelFunc

	// Track if receive loop is running
	running atomic.Bool
}

// NewProtocol creates a new instances of Protocol and returns it.
func NewProtocol(opts ...Option) (*Protocol, error) {
	options := DefaultOptions()
	for _, apply := range opts {
		apply(&options)
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &Protocol{
		transport: options.Transport,
		options:   options,
		pending:   make(map[RequestID]chan RespOrError),
		handlers:  make(map[RequestMethod]RequestHandler),
		notify:    make(map[RequestMethod]NotificationHandler),
		progress:  make(map[ProgressToken]ProgressHandler),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Register default request handlers
	p.RegisterRequestHandler(PingRequestMethod, p.handlePing)
	// Register default notification handlers
	p.RegisterNotificationHandler(CancelledNotificationMethod, p.handleCanceled)
	p.RegisterNotificationHandler(ProgressNotificationMethod, p.handleProgress)

	return p, nil
}

func (p *Protocol) handlePing(_ context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	return &JSONRPCResponse{
		ID:      req.ID,
		Version: JSONRPCVersion,
		Result:  &PingResult{},
	}, nil
}

func (p *Protocol) handleCanceled(ctx context.Context, n *JSONRPCNotification) error {
	cn, ok := n.Notification.(*CancelledNotification)
	if !ok {
		return fmt.Errorf("invalid notification, expected: %q", CancelledNotificationMethod)
	}
	reqID := cn.Params.RequestID
	p.pendingMu.RLock()
	defer p.pendingMu.RUnlock()
	ch, ok := p.pending[reqID]
	if !ok {
		return nil
	}

	msg := "cancel notification received"
	if cn.Params.Reason != nil {
		msg = fmt.Sprintf("caneling request: %s", *cn.Params.Reason)
	}
	errResp := &JSONRPCError{
		ID:      reqID,
		Version: JSONRPCVersion,
		Err: Error{
			Code:    JSONRPCCancelledError,
			Message: msg,
		},
	}
	select {
	case ch <- RespOrError{Err: errResp}:
	case <-time.After(p.options.WaitTimeout):
	case <-p.ctx.Done():
	case <-ctx.Done():
	}
	delete(p.pending, reqID)
	return nil
}

func (p *Protocol) handleProgress(ctx context.Context, n *JSONRPCNotification) error {
	pn, ok := n.Notification.(*ProgressNotification)
	if !ok {
		return fmt.Errorf("invalid notification, expected: %q", ProgressNotificationMethod)
	}
	p.pendingMu.RLock()
	handler, ok := p.progress[pn.Params.ProgressToken]
	p.pendingMu.RUnlock()
	if !ok {
		return nil
	}
	return handler(ctx, pn.Params.Progress, pn.Params.Total)
}

// RegisterRequestHandler registers a handler for the given request method.
// If there is an existing handler registered, this method overrides it.
func (p *Protocol) RegisterRequestHandler(method RequestMethod, handler RequestHandler) {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	p.handlers[method] = handler
}

// DeregisterRequestHandler deregisters a handler for the given request method.
func (p *Protocol) DeregisterRequestHandler(method RequestMethod) {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	delete(p.handlers, method)
}

// RegisterNotificationHandler registers a handler for the given notification request method.
// If there is an existing handler registered, this method overrides it.
func (p *Protocol) RegisterNotificationHandler(method RequestMethod, handler NotificationHandler) {
	p.notifyMu.Lock()
	defer p.notifyMu.Unlock()
	p.notify[method] = handler
}

// DeregisterNotificationHandler deregisters a handler for the given request method.
func (p *Protocol) DeregisterNotificationHandler(method RequestMethod) {
	p.notifyMu.Lock()
	defer p.notifyMu.Unlock()
	delete(p.notify, method)
}

// Connect establishes protocol connection on top of the given transport
// and starts receiving messages sent to it.
func (p *Protocol) Connect() error {
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

	// Start the transport with our protocol context
	if err := p.transport.Start(p.ctx); err != nil {
		p.running.Store(false)
		return fmt.Errorf("failed to start transport: %w", err)
	}

	go p.recvMsg()

	return nil
}

// Close terminates the protocol and its transport.
func (p *Protocol) Close(ctx context.Context) error {
	defer p.running.Store(false)
	defer p.cancel()

	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()

	if p.transport != nil {
		if err := p.transport.Close(); err != nil {
			if !errors.Is(err, ErrTransportClosed) &&
				!errors.Is(err, context.Canceled) {
				return err
			}
		}
	}

	// notify all pending requests
	for reqID, ch := range p.pending {
		errResp := &JSONRPCError{
			ID:      reqID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCConnectionClosed,
				Message: "Connection closed",
			},
		}
		select {
		case ch <- RespOrError{Err: errResp}:
		case <-time.After(p.options.WaitTimeout):
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
		delete(p.pending, reqID)
	}

	return nil
}

// SendRequest sends a request and waits for response
func (p *Protocol) SendRequest(ctx context.Context, req *JSONRPCRequest, opts ...Option) (*JSONRPCResponse, error) {
	options := DefaultOptions()
	for _, apply := range opts {
		apply(&options)
	}

	// Returns the old value and increments
	id := p.nextID.Add(1)
	req.ID = RequestID{Value: id}
	req.Version = JSONRPCVersion

	// Create response channel and register request
	respChan := make(chan RespOrError, 1)

	p.pendingMu.Lock()
	p.pending[req.ID] = respChan
	if options.Progress != nil {
		p.progress[ProgressToken(req.ID)] = options.Progress
	}
	p.pendingMu.Unlock()

	// Clean up on exit
	defer func() {
		p.pendingMu.Lock()
		delete(p.pending, req.ID)
		delete(p.progress, ProgressToken(req.ID))
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
	case <-time.After(p.options.RespTimeout):
		return nil, &JSONRPCError{
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
func (p *Protocol) SendNotification(ctx context.Context, notif *JSONRPCNotification) error {
	notif.Version = JSONRPCVersion
	return p.transport.Send(ctx, notif)
}

// recvMsg handles incoming messages from the transport
func (p *Protocol) recvMsg() {
	defer p.running.Store(false)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			msg, err := p.transport.Receive(p.ctx)
			if err != nil {
				// irrecoverable transport errors
				if errors.Is(err, ErrTransportClosed) ||
					errors.Is(err, context.Canceled) {
					// Transport closed or context cancelled - clean shutdown
					if cErr := p.Close(context.Background()); cErr != nil {
						fmt.Fprintf(os.Stderr, "failed to close protocol: %v", cErr)
					}
					return
				}
				// TODO: maybe we should just shut down completely, too
				// Other errors might be temporary - log and continue
				fmt.Fprintf(os.Stderr, "failed to receive error: %v", err)
				continue
			}
			go p.handleMessage(msg)
		}
	}
}

func (p *Protocol) handleMessage(msg JSONRPCMessage) {
	ctx, cancel := context.WithTimeout(p.ctx, p.options.HandleTimeout)
	defer cancel()

	switch msg.JSONRPCMessageType() {
	case JSONRPCRequestMsgType:
		req, ok := msg.(*JSONRPCRequest)
		if !ok {
			fmt.Fprintf(os.Stderr, "invalid request message type: %T", msg)
			return
		}
		p.handleRequest(ctx, req)

	case JSONRPCNotificationMsgType:
		notif, ok := msg.(*JSONRPCNotification)
		if !ok {
			fmt.Fprintf(os.Stderr, "invalid notification message type: %T", msg)
			return
		}
		p.handleNotification(ctx, notif)

	case JSONRPCResponseMsgType:
		resp, ok := msg.(*JSONRPCResponse)
		if !ok {
			fmt.Fprintf(os.Stderr, "invalid response message type: %T", msg)
			return
		}
		p.handleResponse(ctx, resp)

	case JSONRPCErrorMsgType:
		errMsg, ok := msg.(*JSONRPCError)
		if !ok {
			fmt.Fprintf(os.Stderr, "invalid error message type: %T", msg)
			return
		}
		p.handleError(ctx, errMsg)
	}
}

func (p *Protocol) handleError(ctx context.Context, errMsg *JSONRPCError) {
	p.pendingMu.RLock()
	ch, ok := p.pending[errMsg.ID]
	p.pendingMu.RUnlock()

	if ok {
		select {
		case ch <- RespOrError{Err: errMsg}:
		case <-time.After(p.options.WaitTimeout):
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
	}
}

func (p *Protocol) handleResponse(ctx context.Context, resp *JSONRPCResponse) {
	p.pendingMu.RLock()
	ch, ok := p.pending[resp.ID]
	p.pendingMu.RUnlock()

	if ok {
		select {
		case ch <- RespOrError{Resp: resp}:
		case <-time.After(p.options.WaitTimeout):
		case <-p.ctx.Done():
		case <-ctx.Done():
		}
	}
}

func (p *Protocol) handleNotification(ctx context.Context, n *JSONRPCNotification) {
	p.notifyMu.RLock()
	handler, ok := p.notify[n.Notification.GetMethod()]
	p.notifyMu.RUnlock()

	if ok {
		if err := handler(ctx, n); err != nil {
			fmt.Fprintf(os.Stderr, "notification handler error: %v", err)
		}
	}
}

func (p *Protocol) handleRequest(ctx context.Context, req *JSONRPCRequest) {
	p.handlersMu.RLock()
	handler, ok := p.handlers[req.Request.GetMethod()]
	p.handlersMu.RUnlock()

	if !ok {
		errResp := &JSONRPCError{
			ID:      req.ID,
			Version: JSONRPCVersion,
			Err: Error{
				Code:    JSONRPCMethodNotFoundError,
				Message: "Method not found",
			},
		}
		if err := p.transport.Send(ctx, errResp); err != nil {
			fmt.Fprintf(os.Stderr, "send missing handler error: %v", err)
		}
		return
	}

	resp, err := handler(ctx, req)
	if err != nil {
		errResp := &JSONRPCError{
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
			fmt.Fprintf(os.Stderr, "send handler error: %v", err)
		}
		return
	}

	out := &JSONRPCResponse{
		ID:      req.ID,
		Version: JSONRPCVersion,
		Result:  resp.Result,
	}
	if err := p.transport.Send(ctx, out); err != nil {
		fmt.Fprintf(os.Stderr, "send response error: %v", err)
	}
}
