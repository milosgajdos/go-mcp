package mcp

import (
	"context"
	"fmt"
	"sync"
)

type ClientRequestT[T ID] interface {
	HasReqMethod
	ClientRequest[T]
}

type ClientNotificationT[T ID] interface {
	HasReqMethod
	ClientNotification[T]
}

type Client[T ID, RQ ClientRequestT[T], NF ClientNotificationT[T], RS ClientResult] struct {
	protocol           *Protocol[T, RQ, NF, RS]
	serverVersion      Implementation
	serverCapabilities ServerCapabilities
	mu                 sync.Mutex
}

// NewClient initializes a new MCP client.
func NewClient[
	T ID,
	RQ ClientRequestT[T],
	NF ClientNotificationT[T],
	RS ClientResult,
](transport Transport) (*Client[T, RQ, NF, RS], error) {
	if transport == nil {
		return nil, ErrInvalidTransport
	}

	protocol := NewProtocol[T, RQ, NF, RS](transport)
	return &Client[T, RQ, NF, RS]{
		protocol: protocol,
	}, nil
}

// Connect establishes a connection and initializes the client.
func (c *Client[T, RQ, NF, RS]) Connect(ctx context.Context, clientInfo Implementation) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.protocol.Connect(); err != nil {
		return err
	}

	initReq := InitializeRequest[T]{
		Request: Request[T]{
			Method: InitializeRequestMethod,
		},
		Params: InitializeRequestParams{
			ProtocolVersion: JSONRPCVersion,
			ClientInfo:      clientInfo,
		},
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(initReq).(RQ),
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

	initNotif := InitializedNotification{
		Notification: Notification{
			Method: InitializedNotificationMethod,
		},
	}

	notif := &JSONRPCNotification[T, NF]{
		Notification: any(initNotif).(NF),
		Version:      JSONRPCVersion,
	}
	if err := c.protocol.SendNotification(ctx, notif); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// // assertCapability ensures the server supports a required capability.
// func (c *Client[T, RQ, NF, RS]) assertCapability(capability, method string) {
// 	if _, ok := c.serverCapabilities[capability]; !ok {
// 		panic(fmt.Sprintf("server does not support %s (required for %s)", capability, method))
// 	}
// }

// Ping sends a ping request.
func (c *Client[T, RQ, NF, RS]) Ping(ctx context.Context) error {
	pingReq := PingRequest[T]{
		Request: Request[T]{
			Method: PingRequestMethod,
		},
	}
	req := &JSONRPCRequest[T, RQ]{
		Request: any(pingReq).(RQ),
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// Complete sends a completion request.
func (c *Client[T, RQ, NF, RS]) Complete(ctx context.Context, params CompleteRequestParams) (CompleteResult, error) {
	// c.assertCapability("prompts", CompletionRequestMethod)

	compReq := CompleteRequest[T]{
		Request: Request[T]{
			Method: CompleteRequestMethod,
		},
		Params: params,
	}
	req := &JSONRPCRequest[T, RQ]{
		Request: any(compReq).(RQ),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return CompleteResult{}, err
	}

	return any(resp.Result).(CompleteResult), nil
}

// ListPrompts retrieves a list of available prompts.
func (c *Client[T, RQ, NF, RS]) ListPrompts(ctx context.Context, params *PaginatedRequestParams) (ListPromptsResult, error) {
	// c.assertCapability("prompts", ListPromptsRequestMethod)

	listReq := ListPromptsRequest[T]{
		PaginatedRequest: PaginatedRequest[T]{
			Request: Request[T]{
				Method: ListPromptsRequestMethod,
			},
			Params: params,
		},
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(listReq).(RQ),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return ListPromptsResult{}, err
	}

	return any(resp.Result).(ListPromptsResult), nil
}

// GetPrompt retrieves a specific prompt.
func (c *Client[T, RQ, NF, RS]) GetPrompt(ctx context.Context, params GetPromptRequestParams) (GetPromptResult, error) {
	// c.assertCapability("prompts", GetPromptRequestMethod)

	promptReq := GetPromptRequest[T]{
		Request: Request[T]{
			Method: GetPromptRequestMethod,
		},
		Params: params,
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(promptReq).(RQ),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return GetPromptResult{}, err
	}

	return any(resp.Result).(GetPromptResult), nil
}

// ListResources retrieves a list of resources.
func (c *Client[T, RQ, NF, RS]) ListResources(ctx context.Context, params *PaginatedRequestParams) (ListResourcesResult, error) {
	// c.assertCapability("resources", ListResourcesRequestMethod)

	listReq := ListPromptsRequest[T]{
		PaginatedRequest: PaginatedRequest[T]{
			Request: Request[T]{
				Method: ListResourcesRequestMethod,
			},
			Params: params,
		},
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(listReq).(RQ),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return ListResourcesResult{}, err
	}

	return any(resp.Result).(ListResourcesResult), nil
}

func (c *Client[T, RQ, NF, RS]) ListResourceTemplatesRequest(ctx context.Context, params *PaginatedRequestParams) (ListResourceTemplatesResult, error) {
	// c.assertCapability("resources", ListResourceTemplRequestMethod)

	listReq := ListPromptsRequest[T]{
		PaginatedRequest: PaginatedRequest[T]{
			Request: Request[T]{
				Method: ListResourceTemplRequestMethod,
			},
			Params: params,
		},
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(listReq).(RQ),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return ListResourceTemplatesResult{}, err
	}

	return any(resp.Result).(ListResourceTemplatesResult), nil
}

// ReadResource reads the content of a specific resource.
func (c *Client[T, RQ, NF, RS]) ReadResource(ctx context.Context, params ReadResourceRequestParams) (ReadResourceResult, error) {
	// c.assertCapability("resources", ReadResourceRequestMethod)

	readReq := ReadResourceRequest[T]{
		Request: Request[T]{
			Method: ReadResourceRequestMethod,
		},
		Params: &params,
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(readReq).(RQ),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return ReadResourceResult{}, err
	}

	return any(resp.Result).(ReadResourceResult), nil
}

// CallTool sends a tool call request.
func (c *Client[T, RQ, NF, RS]) CallTool(ctx context.Context, params CallToolRequestParams) (CallToolResult, error) {
	// c.assertCapability("tools", CallToolRequestMethod)

	callReq := CallToolRequest[T]{
		Request: Request[T]{
			Method: CallToolRequestMethod,
		},
		Params: params,
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(callReq).(RQ),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return CallToolResult{}, err
	}

	return any(resp.Result).(CallToolResult), nil
}

// ListTools retrieves a list of available tools.
func (c *Client[T, RQ, NF, RS]) ListTools(ctx context.Context, params *PaginatedRequestParams) (ListToolsResult, error) {
	// c.assertCapability("tools", ListToolsRequestMethod)

	listReq := ListPromptsRequest[T]{
		PaginatedRequest: PaginatedRequest[T]{
			Request: Request[T]{
				Method: ListToolsRequestMethod,
			},
			Params: params,
		},
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(listReq).(RQ),
		Version: JSONRPCVersion,
	}

	resp, err := c.protocol.SendRequest(ctx, req)
	if err != nil {
		return ListToolsResult{}, err
	}

	return any(resp.Result).(ListToolsResult), nil
}

// SubscribeResource subscribes to updates for a specific resource.
func (c *Client[T, RQ, NF, RS]) SubscribeResource(ctx context.Context, params SubscribeRequestParams) error {
	// c.assertCapability("resources", SubscribeRequestMethod)

	subReq := SubscribeRequest[T]{
		Request: Request[T]{
			Method: SubscribeRequestMethod,
		},
		Params: params,
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(subReq).(RQ),
		Version: JSONRPCVersion,
	}
	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// UnsubscribeResource unsubscribes from updates for a specific resource.
func (c *Client[T, RQ, NF, RS]) UnsubscribeResource(ctx context.Context, params UnsubscribeRequestParams) error {
	// c.assertCapability("resources", UnsubscribeRequestMethod)

	unsubReq := UnsubscribeRequest[T]{
		Request: Request[T]{
			Method: UnsubscribeRequestMethod,
		},
		Params: params,
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(unsubReq).(RQ),
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}

// SetLoggingLevel adjusts the logging level on the server.
func (c *Client[T, RQ, NF, RS]) SetLoggingLevel(ctx context.Context, level LoggingLevel) error {
	// c.assertCapability("logging", SetLoggingLevelRequestMethod)

	unsubReq := SetLevelRequest[T]{
		Request: Request[T]{
			Method: SetLevelRequestMethod,
		},
		Params: SetLevelRequestParams{Level: level},
	}

	req := &JSONRPCRequest[T, RQ]{
		Request: any(unsubReq).(RQ),
		Version: JSONRPCVersion,
	}

	_, err := c.protocol.SendRequest(ctx, req)
	return err
}
