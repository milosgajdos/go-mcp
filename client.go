package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

const (
	clientName    = "githuh.com/milosgajdos/go-mcp"
	clientVersion = "v0.unknown"
)

var (
	ErrInvalidResponse = errors.New("invalid JSON-RPC response")
)

// ClientOptions configure client.
type ClientOptions struct {
	EnforceCaps  bool
	Protocol     *Protocol
	Info         Implementation
	Capabilities ClientCapabilities
}

type ClientOption func(*ClientOptions)

// WithClientEnforceCaps enforces server capability checks.
func WithClientEnforceCaps() ClientOption {
	return func(o *ClientOptions) {
		o.EnforceCaps = true
	}
}

// WithClientProtocol configures client Protocol.
func WithClientProtocol(p *Protocol) ClientOption {
	return func(o *ClientOptions) {
		o.Protocol = p
	}
}

// WithClientInfo sets client implementation info.
func WithClientInfo(info Implementation) ClientOption {
	return func(o *ClientOptions) {
		o.Info = info
	}
}

// WithClientCapabilities sets client capabilities.
func WithClientCapabilities(caps ClientCapabilities) ClientOption {
	return func(o *ClientOptions) {
		o.Capabilities = caps
	}
}

// DefaultClientOptions initializes default client options.
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		Info: Implementation{
			Name:    clientName,
			Version: clientVersion,
		},
	}
}

type Client struct {
	options    ClientOptions
	protocol   *Protocol
	caps       ClientCapabilities
	serverInfo Implementation
	serverCaps ServerCapabilities
	connected  atomic.Bool
}

// NewClient initializes a new MCP client.
func NewClient(opts ...ClientOption) (*Client, error) {
	options := DefaultClientOptions()
	for _, apply := range opts {
		apply(&options)
	}

	return &Client{
		protocol: options.Protocol,
		options:  options,
		caps:     options.Capabilities,
	}, nil
}

// GetServerCapabilities returns server capabilities.
func (c *Client) GetServerCapabilities() ServerCapabilities {
	return c.serverCaps
}

// GetServerInfo returns server info.
func (c *Client) GetServerInfo() Implementation {
	return c.serverInfo
}

// Ping sends a ping request.
func (c *Client) Ping(ctx context.Context) error {
	req := &JSONRPCRequest{
		Request: &PingRequest{
			Request: Request{
				Method: PingRequestMethod,
			},
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// Connect establishes a connection and initializes the client.
func (c *Client) Connect(ctx context.Context) error {
	if !c.connected.CompareAndSwap(false, true) {
		return ErrAlreadyConnected
	}

	if err := c.protocol.Connect(); err != nil {
		return fmt.Errorf("connect: %v", err)
	}

	req := &JSONRPCRequest{
		Request: &InitializeRequest{
			Request: Request{
				Method: InitializeRequestMethod,
			},
			Params: InitializeRequestParams{
				ProtocolVersion: LatestVersion,
				ClientInfo:      c.options.Info,
				Capabilities:    c.options.Capabilities,
			},
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("initialize request: %w", err)
	}

	res, ok := resp.Result.(*InitializeResult)
	if !ok {
		return ErrInvalidResponse
	}
	c.serverInfo = res.ServerInfo
	c.serverCaps = copyServerCaps(res.Capabilities)

	notif := &JSONRPCNotification{
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
func (c *Client) Complete(ctx context.Context, params CompleteRequestParams) (*CompleteResult, error) {
	if err := c.assertCaps(CompleteRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest{
		Request: &CompleteRequest{
			Request: Request{
				Method: CompleteRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if res, ok := resp.Result.(*CompleteResult); ok {
		return res, nil
	}

	return nil, ErrInvalidResponse
}

// ListPrompts retrieves a list of available prompts.
func (c *Client) ListPrompts(ctx context.Context, params *PaginatedRequestParams) (*ListPromptsResult, error) {
	if err := c.assertCaps(ListPromptsRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest{
		Request: &ListPromptsRequest{
			Request: Request{
				Method: ListPromptsRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if res, ok := resp.Result.(*ListPromptsResult); ok {
		return res, nil
	}

	return nil, ErrInvalidResponse
}

// GetPrompt retrieves a specific prompt.
func (c *Client) GetPrompt(ctx context.Context, params GetPromptRequestParams) (*GetPromptResult, error) {
	if err := c.assertCaps(GetPromptRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest{
		Request: &GetPromptRequest{
			Request: Request{
				Method: GetPromptRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if res, ok := resp.Result.(*GetPromptResult); ok {
		return res, nil
	}

	return nil, ErrInvalidResponse
}

// ListResources retrieves a list of resources.
func (c *Client) ListResources(ctx context.Context, params *PaginatedRequestParams) (*ListResourcesResult, error) {
	if err := c.assertCaps(ListResourcesRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest{
		Request: &ListPromptsRequest{
			Request: Request{
				Method: ListResourcesRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if res, ok := resp.Result.(*ListResourcesResult); ok {
		return res, nil
	}

	return nil, ErrInvalidResponse
}

func (c *Client) ListResourceTemplatesRequest(ctx context.Context, params *PaginatedRequestParams) (*ListResourceTemplatesResult, error) {
	if err := c.assertCaps(ListResourceTemplatesRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest{
		Request: &ListPromptsRequest{
			Request: Request{
				Method: ListResourceTemplatesRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if res, ok := resp.Result.(*ListResourceTemplatesResult); ok {
		return res, nil
	}

	return nil, ErrInvalidResponse
}

// ReadResource reads the content of a specific resource.
func (c *Client) ReadResource(ctx context.Context, params ReadResourceRequestParams) (*ReadResourceResult, error) {
	if err := c.assertCaps(ReadResourceRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest{
		Request: &ReadResourceRequest{
			Request: Request{
				Method: ReadResourceRequestMethod,
			},
			Params: &params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if res, ok := resp.Result.(*ReadResourceResult); ok {
		return res, nil
	}

	return nil, ErrInvalidResponse
}

// CallTool sends a tool call request.
func (c *Client) CallTool(ctx context.Context, params CallToolRequestParams) (*CallToolResult, error) {
	if err := c.assertCaps(CallToolRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest{
		Request: &CallToolRequest{
			Request: Request{
				Method: CallToolRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if res, ok := resp.Result.(*CallToolResult); ok {
		return res, nil
	}

	return nil, ErrInvalidResponse
}

// ListTools retrieves a list of available tools.
func (c *Client) ListTools(ctx context.Context, params *PaginatedRequestParams) (*ListToolsResult, error) {
	if err := c.assertCaps(ListToolsRequestMethod); err != nil {
		return nil, err
	}

	req := &JSONRPCRequest{
		Request: &ListToolsRequest{
			Request: Request{
				Method: ListToolsRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if res, ok := resp.Result.(*ListToolsResult); ok {
		return res, nil
	}

	return nil, ErrInvalidResponse
}

// SubscribeResource subscribes to updates for a specific resource.
func (c *Client) SubscribeResource(ctx context.Context, params SubscribeRequestParams) error {
	if err := c.assertCaps(SubscribeRequestMethod); err != nil {
		return err
	}

	req := &JSONRPCRequest{
		Request: &SubscribeRequest{
			Request: Request{
				Method: SubscribeRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}
	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// UnsubscribeResource unsubscribes from updates for a specific resource.
func (c *Client) UnsubscribeResource(ctx context.Context, params UnsubscribeRequestParams) error {
	if err := c.assertCaps(UnsubscribeRequestMethod); err != nil {
		return err
	}

	req := &JSONRPCRequest{
		Request: &UnsubscribeRequest{
			Request: Request{
				Method: UnsubscribeRequestMethod,
			},
			Params: params,
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// SetLoggingLevel adjusts the logging level on the server.
func (c *Client) SetLoggingLevel(ctx context.Context, level LoggingLevel) error {
	if err := c.assertCaps(SetLevelRequestMethod); err != nil {
		return err
	}

	req := &JSONRPCRequest{
		Request: &SetLevelRequest{
			Request: Request{
				Method: SetLevelRequestMethod,
			},
			Params: SetLevelRequestParams{Level: level},
		},
		ID:      NewRequestID(uint64(1)),
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

func (c *Client) assertCaps(method RequestMethod) error {
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
