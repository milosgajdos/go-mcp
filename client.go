package mcp

import (
	"context"
	"fmt"
	"sync/atomic"
)

const (
	clientName    = "githuh.com/milosgajdos/go-mcp"
	clientVersion = "v0.unknown"
)

// ClientOptions configure client.
type ClientOptions[T ID] struct {
	EnforceCaps  bool
	Protocol     *Protocol[T]
	Info         Implementation
	Capabilities ClientCapabilities
}

type ClientOption[T ID] func(*ClientOptions[T])

// WithClientEnforceCaps enforces server capability checks.
func WithClientEnforceCaps[T ID]() ClientOption[T] {
	return func(o *ClientOptions[T]) {
		o.EnforceCaps = true
	}
}

// WithClientProtocol configures client Protocol.
func WithClientProtocol[T ID](p *Protocol[T]) ClientOption[T] {
	return func(o *ClientOptions[T]) {
		o.Protocol = p
	}
}

// WithClientInfo sets client implementation info.
func WithClientInfo[T ID](info Implementation) ClientOption[T] {
	return func(o *ClientOptions[T]) {
		o.Info = info
	}
}

// WithClientCapabilities sets client capabilities.
func WithClientCapabilities[T ID](caps ClientCapabilities) ClientOption[T] {
	return func(o *ClientOptions[T]) {
		o.Capabilities = caps
	}
}

// DefaultClientOptions initializes default client options.
func DefaultClientOptions[T ID]() ClientOptions[T] {
	return ClientOptions[T]{
		Info: Implementation{
			Name:    clientName,
			Version: clientVersion,
		},
	}
}

type Client[T ID] struct {
	options    ClientOptions[T]
	protocol   *Protocol[T]
	caps       ClientCapabilities
	serverInfo Implementation
	serverCaps ServerCapabilities
	connected  atomic.Bool
}

// NewClient initializes a new MCP client.
func NewClient[T ID](opts ...ClientOption[T]) (*Client[T], error) {
	options := DefaultClientOptions[T]()
	for _, apply := range opts {
		apply(&options)
	}

	return &Client[T]{
		protocol: options.Protocol,
		options:  options,
		caps:     options.Capabilities,
	}, nil
}

// GetServerCapabilities returns server capabilities.
func (c *Client[T]) GetServerCapabilities() ServerCapabilities {
	return c.serverCaps
}

// GetServerInfo returns server info.
func (c *Client[T]) GetServerInfo() Implementation {
	return c.serverInfo
}

// Ping sends a ping request.
func (c *Client[T]) Ping(ctx context.Context) error {
	req := &JSONRPCRequest[T]{
		Request: &PingRequest[T]{
			Request: Request[T]{
				Method: PingRequestMethod,
			},
		},
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// Connect establishes a connection and initializes the client.
func (c *Client[T]) Connect(ctx context.Context) error {
	if !c.connected.CompareAndSwap(false, true) {
		return ErrAlreadyConnected
	}

	if err := c.protocol.Connect(); err != nil {
		return fmt.Errorf("connect: %v", err)
	}

	req := &JSONRPCRequest[T]{
		Request: &InitializeRequest[T]{
			Request: Request[T]{
				Method: InitializeRequestMethod,
			},
			Params: InitializeRequestParams{
				ProtocolVersion: LatestVersion,
				ClientInfo:      c.options.Info,
				Capabilities:    c.options.Capabilities,
			},
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("initialize request: %w", err)
	}

	res, ok := any(resp.Result).(*InitializeResult)
	if !ok {
		return fmt.Errorf("invalid result")
	}
	c.serverInfo = res.ServerInfo
	c.serverCaps = copyServerCaps(res.Capabilities)

	notif := &JSONRPCNotification[T]{
		Notification: &InitializedNotification{
			Notification: Notification{
				Method: InitializedNotificationMethod,
			},
		},
		Version: JSONRPCVersion,
	}

	if err := c.protocol.SendNotification(ctx, notif); err != nil {
		return fmt.Errorf("initialized notification: %w", err)
	}

	return nil
}

// Complete sends a completion request.
func (c *Client[T]) Complete(ctx context.Context, params CompleteRequestParams) (*CompleteResult, error) {
	if err := c.assertCaps(CompleteRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest[T]{
		Request: &CompleteRequest[T]{
			Request: Request[T]{
				Method: CompleteRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*CompleteResult), nil
}

// ListPrompts retrieves a list of available prompts.
func (c *Client[T]) ListPrompts(ctx context.Context, params *PaginatedRequestParams) (*ListPromptsResult, error) {
	if err := c.assertCaps(ListPromptsRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest[T]{
		Request: &ListPromptsRequest[T]{
			Request: Request[T]{
				Method: ListPromptsRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*ListPromptsResult), nil
}

// GetPrompt retrieves a specific prompt.
func (c *Client[T]) GetPrompt(ctx context.Context, params GetPromptRequestParams) (*GetPromptResult, error) {
	if err := c.assertCaps(GetPromptRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest[T]{
		Request: &GetPromptRequest[T]{
			Request: Request[T]{
				Method: GetPromptRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*GetPromptResult), nil
}

// ListResources retrieves a list of resources.
func (c *Client[T]) ListResources(ctx context.Context, params *PaginatedRequestParams) (*ListResourcesResult, error) {
	if err := c.assertCaps(ListResourcesRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest[T]{
		Request: &ListPromptsRequest[T]{
			Request: Request[T]{
				Method: ListResourcesRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*ListResourcesResult), nil
}

func (c *Client[T]) ListResourceTemplatesRequest(ctx context.Context, params *PaginatedRequestParams) (*ListResourceTemplatesResult, error) {
	if err := c.assertCaps(ListResourceTemplatesRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest[T]{
		Request: &ListPromptsRequest[T]{
			Request: Request[T]{
				Method: ListResourceTemplatesRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*ListResourceTemplatesResult), nil
}

// ReadResource reads the content of a specific resource.
func (c *Client[T]) ReadResource(ctx context.Context, params ReadResourceRequestParams) (*ReadResourceResult, error) {
	if err := c.assertCaps(ReadResourceRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest[T]{
		Request: &ReadResourceRequest[T]{
			Request: Request[T]{
				Method: ReadResourceRequestMethod,
			},
			Params: &params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*ReadResourceResult), nil
}

// CallTool sends a tool call request.
func (c *Client[T]) CallTool(ctx context.Context, params CallToolRequestParams) (*CallToolResult, error) {
	if err := c.assertCaps(CallToolRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest[T]{
		Request: &CallToolRequest[T]{
			Request: Request[T]{
				Method: CallToolRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*CallToolResult), nil
}

// ListTools retrieves a list of available tools.
func (c *Client[T]) ListTools(ctx context.Context, params *PaginatedRequestParams) (*ListToolsResult, error) {
	if err := c.assertCaps(ListToolsRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest[T]{
		Request: &ListToolsRequest[T]{
			Request: Request[T]{
				Method: ListToolsRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return any(resp.Result).(*ListToolsResult), nil
}

// SubscribeResource subscribes to updates for a specific resource.
func (c *Client[T]) SubscribeResource(ctx context.Context, params SubscribeRequestParams) error {
	if err := c.assertCaps(SubscribeRequestMethod); err != nil {
		return err
	}

	req := &JSONRPCRequest[T]{
		Request: &SubscribeRequest[T]{
			Request: Request[T]{
				Method: SubscribeRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}
	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// UnsubscribeResource unsubscribes from updates for a specific resource.
func (c *Client[T]) UnsubscribeResource(ctx context.Context, params UnsubscribeRequestParams) error {
	if err := c.assertCaps(UnsubscribeRequestMethod); err != nil {
		return err
	}

	req := &JSONRPCRequest[T]{
		Request: &UnsubscribeRequest[T]{
			Request: Request[T]{
				Method: UnsubscribeRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// SetLoggingLevel adjusts the logging level on the server.
func (c *Client[T]) SetLoggingLevel(ctx context.Context, level LoggingLevel) error {
	if err := c.assertCaps(SetLevelRequestMethod); err != nil {
		return err
	}

	req := &JSONRPCRequest[T]{
		Request: &SetLevelRequest[T]{
			Request: Request[T]{
				Method: SetLevelRequestMethod,
			},
			Params: SetLevelRequestParams{Level: level},
		},
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

func (c *Client[T]) assertCaps(method RequestMethod) error {
	if !c.options.EnforceCaps {
		return nil
	}
	switch method {
	case SetLevelRequestMethod:
		if len(c.serverCaps.Logging) == 0 {
			return fmt.Errorf("server does not support logging (required by %q)", method)
		}
		return nil
	case GetPromptRequestMethod,
		ListPromptsRequestMethod:
		if c.serverCaps.Prompts == nil {
			return fmt.Errorf("server does not support prompts (required by %q)", method)
		}
		return nil
	case ListResourcesRequestMethod,
		ListResourceTemplatesRequestMethod,
		SubscribeRequestMethod,
		UnsubscribeRequestMethod,
		ReadResourceRequestMethod:
		if c.serverCaps.Resources == nil {
			return fmt.Errorf("server does not support resources (required by %q)", method)
		}
		if method == SubscribeRequestMethod &&
			c.serverCaps.Resources.Subscribe != nil &&
			!*c.serverCaps.Resources.Subscribe {
			return fmt.Errorf("server does not support resource subscription (required by %q)", method)
		}
		return nil
	case ListToolsRequestMethod,
		CallToolRequestMethod:
		if c.serverCaps.Tools == nil {
			return fmt.Errorf("server does not support tools (required by %q)", method)
		}
		return nil
	case CompleteRequestMethod:
		if c.serverCaps.Prompts == nil {
			return fmt.Errorf("server does not support prompts (required by %q)", method)
		}
		return nil
	case InitializeRequestMethod,
		PingRequestMethod:
		return nil
	}
	return nil
}

func copyServerCaps(caps ServerCapabilities) ServerCapabilities {
	experimental := make(ServerCapabilitiesExperimental, len(caps.Experimental))
	for k, v := range caps.Experimental {
		experimental[k] = make(map[string]any, len(v))
		for subKey, subVal := range v {
			experimental[k][subKey] = subVal
		}
	}

	logging := make(ServerCapabilitiesLogging, len(caps.Logging))
	for k, v := range caps.Logging {
		logging[k] = v
	}

	return ServerCapabilities{
		Experimental: experimental,
		Logging:      logging,
		Prompts:      caps.Prompts, // Safe as Prompts is already a pointer
		Resources:    caps.Resources,
		Tools:        caps.Tools,
	}
}
