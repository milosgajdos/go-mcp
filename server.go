package mcp

import (
	"context"
	"fmt"
	"sync/atomic"
)

const (
	serverName    = "githuh.com/milosgajdos/go-mcp"
	serverVersion = "v1.alpha"
)

type ServerOptions struct {
	EnforceCaps  bool
	Transport    Transport
	Info         Implementation
	Capabilities ServerCapabilities
}

type ServerOption func(*ServerOptions)

func WithServerEnforceCaps() ServerOption {
	return func(o *ServerOptions) {
		o.EnforceCaps = true
	}
}

func WithServerTransport(tr Transport) ServerOption {
	return func(o *ServerOptions) {
		o.Transport = tr
	}
}

func WithServerInfo(info Implementation) ServerOption {
	return func(o *ServerOptions) {
		o.Info = info
	}
}
func WithServerCapabilities(caps ServerCapabilities) ServerOption {
	return func(o *ServerOptions) {
		o.Capabilities = caps
	}
}

func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		Info: Implementation{
			Name:    serverName,
			Version: serverVersion,
		},
	}
}

type Server[T ID] struct {
	options    ServerOptions
	protocol   *Protocol[T]
	caps       ServerCapabilities
	clientInfo Implementation
	clientCaps ClientCapabilities
	connected  atomic.Bool
}

// NewServer initializes a new MCP server.
func NewServer[T ID](opts ...ServerOption) (*Server[T], error) {
	options := ServerOptions{}
	for _, apply := range opts {
		apply(&options)
	}

	// NOTE: consider using In-memory transport as default
	if options.Transport == nil {
		return nil, ErrInvalidTransport
	}

	protocol := NewProtocol[T](WithTransport(options.Transport))
	server := &Server[T]{
		options:  options,
		protocol: protocol,
		caps:     options.Capabilities,
	}

	// Register core handlers
	protocol.RegisterRequestHandler(InitializeRequestMethod, server.handleInitialize)
	protocol.RegisterRequestHandler(PingRequestMethod, server.handlePing)
	protocol.RegisterNotificationHandler(InitializedNotificationMethod, server.handleInitialized)

	return server, nil
}

// Connect establishes server transport.
func (s *Server[T]) Connect(context.Context) error {
	if !s.connected.CompareAndSwap(false, true) {
		return ErrAlreadyConnected
	}
	return s.protocol.Connect()
}

// Close terminates the server.
func (s *Server[T]) Close(ctx context.Context) error {
	return s.protocol.Close(ctx)
}

// GetClientCapabilities returns client capabilities.
func (s *Server[T]) GetClientCapabilities() ClientCapabilities {
	return s.clientCaps
}

// GetClientInfo returns client info.
func (s *Server[T]) GetClientInfo() Implementation {
	return s.clientInfo
}

// HandleRequest registers a requset handler for the given request method.
func (s *Server[T]) HandleRequest(method RequestMethod, handler RequestHandler[T]) error {
	if err := s.assertReqCaps(method); err != nil {
		return err
	}
	s.protocol.RegisterRequestHandler(method, handler)
	return nil
}

// HandleNotification registers a notification handler for the given notification method.
func (s *Server[T]) HandleNotification(method RequestMethod, handler NotificationHandler[T]) error {
	if err := s.assertNotifCaps(method); err != nil {
		return err
	}
	s.protocol.RegisterNotificationHandler(method, handler)
	return nil
}

func (s *Server[T]) handleInitialize(_ context.Context, req *JSONRPCRequest[T]) (*JSONRPCResponse[T], error) {
	var params InitializeRequestParams
	if req.Request == nil {
		return nil, fmt.Errorf("empty request")
	}

	initReq, ok := req.Request.(*InitializeRequest[T])
	if !ok {
		return nil, fmt.Errorf("invalid initialize request type: %T", req.Request)
	}

	params = initReq.Params
	s.clientInfo = params.ClientInfo
	s.clientCaps = params.Capabilities

	requestedVersion := LatestVersion
	if IsSupportedVersion(params.ProtocolVersion) {
		requestedVersion = params.ProtocolVersion
	}

	return &JSONRPCResponse[T]{
		Result: &InitializeResult{
			Result:          Result{},
			ProtocolVersion: requestedVersion,
			Capabilities:    s.options.Capabilities,
			ServerInfo:      s.options.Info,
		},
		ID:      req.ID,
		Version: JSONRPCVersion,
	}, nil
}

func (s *Server[T]) handlePing(_ context.Context, req *JSONRPCRequest[T]) (*JSONRPCResponse[T], error) {
	return &JSONRPCResponse[T]{
		ID:      req.ID,
		Version: JSONRPCVersion,
		Result:  &PingResult{},
	}, nil
}

func (s *Server[T]) handleInitialized(context.Context, *JSONRPCNotification[T]) error {
	// Server is now fully initialized
	// we could trigger any post-initialization logic here
	return nil
}

// SendLoggingMessage sends a logging message to the client
func (s *Server[T]) SendLoggingMessage(ctx context.Context, params LoggingMessageNotificationParams) error {
	if err := s.assertNotifCaps(LoggingMessageNotificationMethod); err != nil {
		return err
	}

	notif := &JSONRPCNotification[T]{
		Notification: &LoggingMessageNotification{
			Notification: Notification{
				Method: LoggingMessageNotificationMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	return s.protocol.SendNotification(ctx, notif)
}

// SendResourceUpdated notifies client about resource updates
func (s *Server[T]) SendResourceUpdated(ctx context.Context, params ResourceUpdatedNotificationParams) error {
	if err := s.assertNotifCaps(ResourceUpdatedNotificationMethod); err != nil {
		return err
	}

	notif := &JSONRPCNotification[T]{
		Notification: &ResourceUpdatedNotification{
			Notification: Notification{
				Method: ResourceUpdatedNotificationMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	return s.protocol.SendNotification(ctx, notif)
}

// SendResourceListChanged notifies client about resource list changes
func (s *Server[T]) SendResourceListChanged(ctx context.Context) error {
	if err := s.assertNotifCaps(ResourceListChangedNotificationMethod); err != nil {
		return err
	}

	notif := &JSONRPCNotification[T]{
		Notification: &ResourceListChangedNotification{
			Notification: Notification{
				Method: ResourceListChangedNotificationMethod,
			},
		},
		Version: JSONRPCVersion,
	}

	return s.protocol.SendNotification(ctx, notif)
}

// SendToolListChanged notifies client about tool list changes
func (s *Server[T]) SendToolListChanged(ctx context.Context) error {
	if err := s.assertNotifCaps(ToolListChangedNotificationMethod); err != nil {
		return err
	}

	notif := &JSONRPCNotification[T]{
		Notification: &ToolListChangedNotification{
			Notification: Notification{
				Method: ToolListChangedNotificationMethod,
			},
		},
		Version: JSONRPCVersion,
	}

	return s.protocol.SendNotification(ctx, notif)
}

// SendPromptListChanged notifies client about prompt list changes
func (s *Server[T]) SendPromptListChanged(ctx context.Context) error {
	if err := s.assertNotifCaps(PromptListChangedNotificationMethod); err != nil {
		return err
	}

	notif := &JSONRPCNotification[T]{
		Notification: &PromptListChangedNotification{
			Notification: Notification{
				Method: PromptListChangedNotificationMethod,
			},
		},
		Version: JSONRPCVersion,
	}

	return s.protocol.SendNotification(ctx, notif)
}

// Ping sends a ping request.
func (s *Server[T]) Ping(ctx context.Context) error {
	if err := s.assertClientCaps(PingRequestMethod); err != nil {
		return err
	}
	req := &JSONRPCRequest[T]{
		Request: &PingRequest[T]{
			Request: Request[T]{
				Method: PingRequestMethod,
			},
		},
		Version: JSONRPCVersion,
	}

	_, err := s.protocol.SendRequest(ctx, req)
	return err
}

// CreateMessage sends a create message request.
func (s *Server[T]) CreateMessage(ctx context.Context, params CreateMessageRequestParams) (*CreateMessageResult, error) {
	if err := s.assertClientCaps(CreateMessageRequestMethod); err != nil {
		return nil, err
	}
	req := &JSONRPCRequest[T]{
		Request: &CreateMessageRequest[T]{
			Request: Request[T]{
				Method: CreateMessageRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := s.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*CreateMessageResult), nil
}

// ListRoots sends a list roots request.
func (s *Server[T]) ListRoots(ctx context.Context) (*ListRootsResult, error) {
	if err := s.assertClientCaps(ListRootsRequestMethod); err != nil {
		return nil, err
	}
	req := &JSONRPCRequest[T]{
		Request: &ListRootsRequest[T]{
			Request: Request[T]{
				Method: ListRootsRequestMethod,
			},
		},
		Version: JSONRPCVersion,
	}

	resp, err := s.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*ListRootsResult), nil
}

func (s *Server[T]) assertClientCaps(method RequestMethod) error {
	if !s.options.EnforceCaps {
		return nil
	}
	switch method {
	case CreateMessageRequestMethod:
		if len(s.clientCaps.Sampling) == 0 {
			return fmt.Errorf("client does not support sampling (required by %q)", method)
		}
		return nil
	case ListRootsRequestMethod:
		if s.clientCaps.Roots == nil {
			return fmt.Errorf("client does not support list roots (required by %q)", method)
		}
		return nil
	case PingRequestMethod:
		return nil
	}
	return nil
}

func (s *Server[T]) assertReqCaps(method RequestMethod) error {
	switch method {
	case SetLevelRequestMethod:
		if len(s.caps.Logging) == 0 {
			return fmt.Errorf("server does not support logging (required by %q)", method)
		}
		return nil
	case ListPromptsRequestMethod,
		GetPromptRequestMethod:
		if s.caps.Prompts == nil {
			return fmt.Errorf("server does not support prompts (required by %q)", method)
		}
		return nil
	case ListResourcesRequestMethod,
		ListResourceTemplatesRequestMethod,
		ReadResourceRequestMethod:
		if s.caps.Resources == nil {
			return fmt.Errorf("server does not support resources (required by %q)", method)
		}
		return nil
	case ListToolsRequestMethod,
		CallToolRequestMethod:
		if s.caps.Tools == nil {
			return fmt.Errorf("server does not support tools (required by %q)", method)
		}
		return nil
	case InitializeRequestMethod,
		PingRequestMethod:
		return nil
	}
	return nil
}

func (s *Server[T]) assertNotifCaps(method RequestMethod) error {
	switch method {
	case LoggingMessageNotificationMethod:
		if len(s.caps.Logging) == 0 {
			return fmt.Errorf("server does not support logging (required by %q)", method)
		}
	case ResourceUpdatedNotificationMethod,
		ResourceListChangedNotificationMethod:
		if s.caps.Resources == nil {
			return fmt.Errorf("server does not support resources (required by %q)", method)
		}
	case ToolListChangedNotificationMethod:
		if s.caps.Tools == nil {
			return fmt.Errorf("server does not support tools (required by %q)", method)
		}
	case PromptListChangedNotificationMethod:
		if s.caps.Prompts == nil {
			return fmt.Errorf("server does not support prompts (required by %q)", method)
		}
	case CancelledNotificationMethod,
		ProgressNotificationMethod:
		return nil
	}
	return nil
}
