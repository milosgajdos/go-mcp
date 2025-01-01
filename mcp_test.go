package mcp

import (
	"encoding/json"
	"testing"
)

func TestRequestMessages(t *testing.T) {
	tests := []struct {
		name    string
		request JSONRPCRequest[uint64]
	}{
		{
			name: "PingRequest",
			request: JSONRPCRequest[uint64]{
				Request: &PingRequest[uint64]{
					Request: Request[uint64]{
						Method: PingRequestMethod,
					},
					Params: &PingRequestParams[uint64]{
						AdditionalProperties: map[string]any{
							"foo": "bar",
						},
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "InitializeRequest",
			request: JSONRPCRequest[uint64]{
				Request: &InitializeRequest[uint64]{
					Request: Request[uint64]{
						Method: InitializeRequestMethod,
					},
					Params: InitializeRequestParams{
						ClientInfo: Implementation{
							Name:    "myclient",
							Version: "v2",
						},
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CompleteRequest",
			request: JSONRPCRequest[uint64]{
				Request: &CompleteRequest[uint64]{
					Request: Request[uint64]{
						Method: CompleteRequestMethod,
					},
					Params: CompleteRequestParams{
						Ref: CompleteRequestParamsRef{
							PromptReference: &PromptReference{
								Type: PromptReferenceType,
								Name: "SomeRefType",
							},
						},
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "SetLevelRequest",
			request: JSONRPCRequest[uint64]{
				Request: &SetLevelRequest[uint64]{
					Request: Request[uint64]{
						Method: SetLevelRequestMethod,
					},
					Params: SetLevelRequestParams{
						Level: LoggingLevelEmergency,
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "GetPromptRequest",
			request: JSONRPCRequest[uint64]{
				Request: &GetPromptRequest[uint64]{
					Request: Request[uint64]{
						Method: GetPromptRequestMethod,
					},
					Params: GetPromptRequestParams{
						Name: "foo",
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListPromptsRequest",
			request: JSONRPCRequest[uint64]{
				Request: &ListPromptsRequest[uint64]{
					Request: Request[uint64]{
						Method: ListPromptsRequestMethod,
					},
					Params: &PaginatedRequestParams{
						Cursor: &[]Cursor{"test-cursor"}[0],
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListResourcesRequest",
			request: JSONRPCRequest[uint64]{
				Request: &ListResourcesRequest[uint64]{
					Request: Request[uint64]{
						Method: ListResourcesRequestMethod,
					},
					Params: &PaginatedRequestParams{
						Cursor: &[]Cursor{"test-cursor"}[0],
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListResourceTemplatesRequest",
			request: JSONRPCRequest[uint64]{
				Request: &ListResourceTemplatesRequest[uint64]{
					Request: Request[uint64]{
						Method: ListResourceTemplatesRequestMethod,
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListRootsRequest",
			request: JSONRPCRequest[uint64]{
				Request: &ListRootsRequest[uint64]{
					Request: Request[uint64]{
						Method: ListRootsRequestMethod,
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ReadResourceRequest",
			request: JSONRPCRequest[uint64]{
				Request: &ReadResourceRequest[uint64]{
					Request: Request[uint64]{
						Method: ReadResourceRequestMethod,
					},
					Params: &ReadResourceRequestParams{
						URI: "foo://bar",
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "SubscribeRequest",
			request: JSONRPCRequest[uint64]{
				Request: &SubscribeRequest[uint64]{
					Request: Request[uint64]{
						Method: SubscribeRequestMethod,
					},
					Params: SubscribeRequestParams{
						URI: "foo://bar",
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "UnsubscribeRequest",
			request: JSONRPCRequest[uint64]{
				Request: &UnsubscribeRequest[uint64]{
					Request: Request[uint64]{
						Method: SubscribeRequestMethod,
					},
					Params: UnsubscribeRequestParams{
						URI: "foo://bar",
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CallToolRequest",
			request: JSONRPCRequest[uint64]{
				Request: &CallToolRequest[uint64]{
					Request: Request[uint64]{
						Method: CallToolRequestMethod,
					},
					Params: CallToolRequestParams{
						Name: "CallToolName",
						Arguments: CallToolRequestParamsArguments{
							"arg1": "one",
						},
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CreateMessageRequest",
			request: JSONRPCRequest[uint64]{
				Request: &CreateMessageRequest[uint64]{
					Request: Request[uint64]{
						Method: CreateMessageRequestMethod,
					},
					Params: CreateMessageRequestParams{
						Messages: []SamplingMessage{
							{
								Role: RoleUser,
								Content: SamplingMessageContent{
									TextContent: &TextContent{
										Text: "Hello world",
										Type: TextContentType,
									},
								},
							},
						},
						ModelPreferences: &ModelPreferences{
							SpeedPriority: &[]float64{0.1}[0],
							Hints: []ModelHint{
								{
									Name: &[]string{"modelName"}[0],
								},
							},
						},
						MaxTokens: 100,
					},
				},
				ID:      RequestID[uint64]{Value: 2},
				Version: JSONRPCVersion,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Errorf("Marshal %s failed: %v", tt.name, err)
			}

			// Test unmarshaling
			var decoded JSONRPCRequest[uint64]
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Errorf("Unmarshal %s failed: %v", tt.name, err)
			}

			// Test that the decoded request matches the original
			reencoded, err := json.Marshal(decoded)
			if err != nil {
				t.Errorf("Re-marshal of decoded %s failed: %v", tt.name, err)
			}

			if string(data) != string(reencoded) {
				t.Errorf("Re-encoded %s doesn't match original.\nOriginal: %s\nRe-encoded: %s",
					tt.name, string(data), string(reencoded))
			}
		})
	}
}

func TestNotificationMessages(t *testing.T) {
	tests := []struct {
		name         string
		notification JSONRPCNotification[uint64]
	}{
		{
			name: "ProgressNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &ProgressNotification[uint64]{
					Notification: Notification{
						Method: ProgressNotificationMethod,
					},
					Params: ProgressNotificationParams[uint64]{
						Progress: 0.6,
						ProgressToken: ProgressToken[uint64]{
							Value: 100,
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "InitializedNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &InitializedNotification{
					Notification: Notification{
						Method: InitializedNotificationMethod,
					},
					Params: &InitializedNotificationParams{
						Meta: InitializedNotificationParamsMeta{
							"bar": "baz",
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "RootsListChangedNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &RootsListChangedNotification{
					Notification: Notification{
						Method: RootsListChangedNotificationMethod,
						Params: &NotificationParams{
							Meta: NotificationParamsMeta{
								"bar": "baz",
							},
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "LoggingMessageNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &LoggingMessageNotification{
					Notification: Notification{
						Method: LoggingMessageNotificationMethod,
					},
					Params: LoggingMessageNotificationParams{
						Level: LoggingLevelAlert,
						Data:  "some data to log",
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ResourceUpdatedNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &ResourceUpdatedNotification{
					Notification: Notification{
						Method: ResourceUpdatedNotificationMethod,
					},
					Params: ResourceUpdatedNotificationParams{
						URI: "test://resource",
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ResourceListChangedNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &ResourceListChangedNotification{
					Notification: Notification{
						Method: ResourceListChangedNotificationMethod,
						Params: &NotificationParams{
							Meta: NotificationParamsMeta{
								"bar": "baz",
							},
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ToolListChangedNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &ToolListChangedNotification{
					Notification: Notification{
						Method: ToolListChangedNotificationMethod,
						Params: &NotificationParams{
							Meta: NotificationParamsMeta{
								"bar": "baz",
							},
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "PromptListChangedNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &PromptListChangedNotification{
					Notification: Notification{
						Method: PromptListChangedNotificationMethod,
						Params: &NotificationParams{
							Meta: NotificationParamsMeta{
								"bar": "baz",
							},
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CancelledNotification",
			notification: JSONRPCNotification[uint64]{
				Notification: &CancelledNotification[uint64]{
					Notification: Notification{
						Method: CancelledNotificationMethod,
					},
					Params: CancelledNotificationParams[uint64]{
						Reason: &[]string{"some reason"}[0],
					},
				},
				Version: JSONRPCVersion,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.notification)
			if err != nil {
				t.Errorf("Marshal %s failed: %v", tt.name, err)
			}

			// Test unmarshaling
			var decoded JSONRPCNotification[uint64]
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Errorf("Unmarshal %s failed: %v", tt.name, err)
			}

			// Test that the decoded notification matches the original
			reencoded, err := json.Marshal(decoded)
			if err != nil {
				t.Errorf("Re-marshal of decoded %s failed: %v", tt.name, err)
			}

			if string(data) != string(reencoded) {
				t.Errorf("Re-encoded %s doesn't match original.\nOriginal: %s\nRe-encoded: %s",
					tt.name, string(data), string(reencoded))
			}
		})
	}
}

func TestResponseMessages(t *testing.T) {
	tests := []struct {
		name     string
		response JSONRPCResponse[uint64]
	}{
		{
			name: "CreateMessageResult",
			response: JSONRPCResponse[uint64]{
				Result: &CreateMessageResult{
					Result: Result{
						Meta: ResultMeta{
							"msg": "result",
						},
					},
					Model: "foobarModel",
					Role:  RoleUser,
					Content: SamplingMessageContent{
						TextContent: &TextContent{
							Text: "someTextContent",
							Type: TextContentType,
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListRootsResult",
			response: JSONRPCResponse[uint64]{
				Result: &ListRootsResult{
					Result: Result{
						Meta: ResultMeta{
							"msg": "listRootsRes",
						},
					},
					Roots: []Root{
						{URI: "fo://bar"},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CompleteResult",
			response: JSONRPCResponse[uint64]{
				Result: &CompleteResult{
					Result: Result{
						Meta: ResultMeta{
							"msg": "CompleteResult",
						},
					},
					Completion: CompleteResultCompletion{
						Values:  []string{"option1", "option2"},
						HasMore: &[]bool{true}[0],
						Total:   &[]int{5}[0],
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "InitializeResult",
			response: JSONRPCResponse[uint64]{
				Result: &InitializeResult{
					Result: Result{
						Meta: ResultMeta{
							"msg": "InitializeResult",
						},
					},
					Capabilities: ServerCapabilities{
						Logging: ServerCapabilitiesLogging{
							"log": "something",
						},
					},
					ProtocolVersion: "foo",
					ServerInfo: Implementation{
						Name:    "name",
						Version: "version",
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "GetPromptResult",
			response: JSONRPCResponse[uint64]{
				Result: &GetPromptResult{
					Result: Result{
						Meta: ResultMeta{
							"msg": "GetPromptResult",
						},
					},
					Description: &[]string{"A test prompt"}[0],
					Messages: []PromptMessage{
						{
							Role: RoleUser,
							Content: PromptMessageContent{
								TextContent: &TextContent{
									Type: TextContentType,
									Text: "Hello world",
								},
							},
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListPromptsResult",
			response: JSONRPCResponse[uint64]{
				Result: &ListPromptsResult{
					PaginatedResult: PaginatedResult{
						Meta: ResultMeta{
							"msg": "ListPromptsResult",
						},
						NextCursor: &[]Cursor{"next-page"}[0],
					},
					Prompts: []Prompt{
						{
							Name:        "test-prompt",
							Description: &[]string{"A test prompt"}[0],
							Arguments: []PromptArgument{
								{
									Name:        "arg1",
									Description: &[]string{"First argument"}[0],
									Required:    &[]bool{true}[0],
								},
							},
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListResourcesResult",
			response: JSONRPCResponse[uint64]{
				Result: &ListResourcesResult{
					PaginatedResult: PaginatedResult{
						Meta: ResultMeta{
							"msg": "ListResourcesResult",
						},
						NextCursor: &[]Cursor{"next-page"}[0],
					},
					Resources: []Resource{
						{
							Name:        "test-resource",
							URI:         "test://resource",
							Description: &[]string{"A test resource"}[0],
							MimeType:    &[]string{"text/plain"}[0],
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListResourceTemplatesResult",
			response: JSONRPCResponse[uint64]{
				Result: &ListResourceTemplatesResult{
					PaginatedResult: PaginatedResult{
						Meta: ResultMeta{
							"msg": "ListResourceTemplatesResult",
						},
						NextCursor: &[]Cursor{"next-page"}[0],
					},
					ResourceTemplates: []ResourceTemplate{
						{
							Name:        "test-template",
							URITemplate: "test://{param}",
							Description: &[]string{"A test template"}[0],
							MimeType:    &[]string{"text/plain"}[0],
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ReadResourceResult",
			response: JSONRPCResponse[uint64]{
				Result: &ReadResourceResult{
					Result: Result{
						Meta: ResultMeta{
							"msg": "ReadResourceResult",
						},
					},
					Contents: []ReadResourceResultContent{
						{
							TextResourceContents: &TextResourceContents{
								ResourceContents: ResourceContents{
									URI:      "test://resource",
									MimeType: &[]string{"text/plain"}[0],
								},
								Text: "Hello, World!",
							},
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CallToolResult",
			response: JSONRPCResponse[uint64]{
				Result: &CallToolResult{
					Result: Result{
						Meta: ResultMeta{
							"msg": "CallToolResult",
						},
					},
					Content: []CallToolResultContent{
						{
							TextContent: &TextContent{
								Type: TextContentType,
								Text: "Tool execution result",
							},
						},
					},
					IsError: &[]bool{false}[0],
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListToolsResult",
			response: JSONRPCResponse[uint64]{
				Result: &ListToolsResult{
					PaginatedResult: PaginatedResult{
						Meta: ResultMeta{
							"msg": "ListToolsResult",
						},
						NextCursor: &[]Cursor{"next-page"}[0],
					},
					Tools: []Tool{
						{
							Name:        "test-tool",
							Description: &[]string{"A test tool"}[0],
							InputSchema: ToolInputSchema{
								Type: ObjectType,
								Properties: ToolInputSchemaProperties{
									"arg1": {
										"type":        "string",
										"description": "First argument",
									},
								},
							},
						},
					},
				},
				Version: JSONRPCVersion,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Errorf("Marshal %s failed: %v", tt.name, err)
			}

			// Test unmarshaling
			var decoded JSONRPCResponse[uint64]
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Errorf("Unmarshal %s failed: %v", tt.name, err)
			}

			// Test that the decoded response matches the original
			reencoded, err := json.Marshal(decoded)
			if err != nil {
				t.Errorf("Re-marshal of decoded %s failed: %v", tt.name, err)
			}

			if string(data) != string(reencoded) {
				t.Errorf("Re-encoded %s doesn't match original.\nOriginal: %s\nRe-encoded: %s",
					tt.name, string(data), string(reencoded))
			}
		})
	}
}

func TestJSONRPCErrorMessages(t *testing.T) {
	tests := []struct {
		name  string
		error JSONRPCError[uint64]
	}{
		{
			name: "MethodNotFoundError",
			error: JSONRPCError[uint64]{
				ID:      RequestID[uint64]{Value: 3},
				Version: JSONRPCVersion,
				Err: Error{
					Code:    JSONRPCMethodNotFoundError,
					Message: "Method not found",
				},
			},
		},
		{
			name: "InvalidParamsError",
			error: JSONRPCError[uint64]{
				ID:      RequestID[uint64]{Value: 10},
				Version: JSONRPCVersion,
				Err: Error{
					Code:    JSONRPCInvalidParamError,
					Message: "Invalid parameters",
				},
			},
		},
		{
			name: "InternalError",
			error: JSONRPCError[uint64]{
				ID:      RequestID[uint64]{Value: 7},
				Version: JSONRPCVersion,
				Err: Error{
					Code:    JSONRPCInternalError,
					Message: "Internal error occurred",
				},
			},
		},
		{
			name: "CustomError",
			error: JSONRPCError[uint64]{
				ID:      RequestID[uint64]{Value: 1},
				Version: JSONRPCVersion,
				Err: Error{
					Code:    -32000,
					Message: "Custom server error",
					Data: map[string]any{
						"detail": "Additional error details",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.error)
			if err != nil {
				t.Errorf("Marshal %s failed: %v", tt.name, err)
			}

			// Test unmarshaling
			var decoded JSONRPCError[uint64]
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Errorf("Unmarshal %s failed: %v", tt.name, err)
			}

			// Test that the decoded error matches the original
			reencoded, err := json.Marshal(decoded)
			if err != nil {
				t.Errorf("Re-marshal of decoded %s failed: %v", tt.name, err)
			}

			if string(data) != string(reencoded) {
				t.Errorf("Re-encoded %s doesn't match original.\nOriginal: %s\nRe-encoded: %s",
					tt.name, string(data), string(reencoded))
			}
		})
	}
}
