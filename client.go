package mcp

import (
	"context"
	"fmt"
	"sync"
)

type Client[T ID] struct {
	protocol           *Protocol[T]
	serverVersion      Implementation
	serverCapabilities ServerCapabilities
	mu                 sync.Mutex
}

// NewClient initializes a new MCP client.
func NewClient[T ID](transport Transport) (*Client[T], error) {
	if transport == nil {
		return nil, ErrInvalidTransport
	}

	protocol := NewProtocol[T](transport)
	return &Client[T]{
		protocol: protocol,
	}, nil
}

// Connect establishes a connection and initializes the client.
func (c *Client[T]) Connect(ctx context.Context, clientInfo Implementation) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.protocol.Connect(); err != nil {
		return err
	}

	req := &JSONRPCRequest[T]{
		Request: &InitializeRequest[T]{
			Request: Request[T]{
				Method: InitializeRequestMethod,
			},
			Params: InitializeRequestParams{
				ProtocolVersion: LatestVersion,
				ClientInfo:      clientInfo,
			},
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	res, ok := any(resp.Result).(InitializeResult)
	if !ok {
		return fmt.Errorf("invalid result")
	}
	c.serverCapabilities = res.Capabilities
	c.serverVersion = res.ServerInfo

	notif := &JSONRPCNotification[T]{
		Notification: &InitializedNotification{
			Notification: Notification{
				Method: InitializedNotificationMethod,
			},
		},
		Version: JSONRPCVersion,
	}

	if err := c.protocol.SendNotification(ctx, notif); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// // assertCapability ensures the server supports a required capability.
// func (c *Client[T]) assertCapability(capability, method string) {
// 	if _, ok := c.serverCapabilities[capability]; !ok {
// 		panic(fmt.Sprintf("server does not support %s (required for %s)", capability, method))
// 	}
// }

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

// Complete sends a completion request.
func (c *Client[T]) Complete(ctx context.Context, params CompleteRequestParams) (CompleteResult, error) {
	// c.assertCapability("prompts", CompletionRequestMethod)

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
		return CompleteResult{}, err
	}

	return any(resp.Result).(CompleteResult), nil
}

// ListPrompts retrieves a list of available prompts.
func (c *Client[T]) ListPrompts(ctx context.Context, params *PaginatedRequestParams) (ListPromptsResult, error) {
	// c.assertCapability("prompts", ListPromptsRequestMethod)

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
		return ListPromptsResult{}, err
	}

	return any(resp.Result).(ListPromptsResult), nil
}

// GetPrompt retrieves a specific prompt.
func (c *Client[T]) GetPrompt(ctx context.Context, params GetPromptRequestParams) (GetPromptResult, error) {
	// c.assertCapability("prompts", GetPromptRequestMethod)

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
		return GetPromptResult{}, err
	}

	return any(resp.Result).(GetPromptResult), nil
}

// ListResources retrieves a list of resources.
func (c *Client[T]) ListResources(ctx context.Context, params *PaginatedRequestParams) (ListResourcesResult, error) {
	// c.assertCapability("resources", ListResourcesRequestMethod)

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
		return ListResourcesResult{}, err
	}

	return any(resp.Result).(ListResourcesResult), nil
}

func (c *Client[T]) ListResourceTemplatesRequest(ctx context.Context, params *PaginatedRequestParams) (ListResourceTemplatesResult, error) {
	// c.assertCapability("resources", ListResourceTemplRequestMethod)

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
		return ListResourceTemplatesResult{}, err
	}

	return any(resp.Result).(ListResourceTemplatesResult), nil
}

// ReadResource reads the content of a specific resource.
func (c *Client[T]) ReadResource(ctx context.Context, params ReadResourceRequestParams) (ReadResourceResult, error) {
	// c.assertCapability("resources", ReadResourceRequestMethod)

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
		return ReadResourceResult{}, err
	}

	return any(resp.Result).(ReadResourceResult), nil
}

// CallTool sends a tool call request.
func (c *Client[T]) CallTool(ctx context.Context, params CallToolRequestParams) (CallToolResult, error) {
	// c.assertCapability("tools", CallToolRequestMethod)

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
		return CallToolResult{}, err
	}

	return any(resp.Result).(CallToolResult), nil
}

// ListTools retrieves a list of available tools.
func (c *Client[T]) ListTools(ctx context.Context, params *PaginatedRequestParams) (ListToolsResult, error) {
	// c.assertCapability("tools", ListToolsRequestMethod)

	req := &JSONRPCRequest[T]{
		Request: &ListPromptsRequest[T]{
			Request: Request[T]{
				Method: ListToolsRequestMethod,
			},
			Params: params,
		},
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return ListToolsResult{}, err
	}

	return any(resp.Result).(ListToolsResult), nil
}

// SubscribeResource subscribes to updates for a specific resource.
func (c *Client[T]) SubscribeResource(ctx context.Context, params SubscribeRequestParams) error {
	// c.assertCapability("resources", SubscribeRequestMethod)

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
	// c.assertCapability("resources", UnsubscribeRequestMethod)

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
	// c.assertCapability("logging", SetLoggingLevelRequestMethod)

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
