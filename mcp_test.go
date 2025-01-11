package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRequestMessages(t *testing.T) {
	tests := []struct {
		name    string
		request JSONRPCRequest
	}{
		{
			name: "PingRequest",
			request: JSONRPCRequest{
				Request: &PingRequest{
					Request: Request{
						Method: PingRequestMethod,
					},
					Params: &PingRequestParams{
						AdditionalProperties: map[string]any{
							"foo": "bar",
						},
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "InitializeRequest",
			request: JSONRPCRequest{
				Request: &InitializeRequest{
					Request: Request{
						Method: InitializeRequestMethod,
					},
					Params: InitializeRequestParams{
						ClientInfo: Implementation{
							Name:    "myclient",
							Version: "v2",
						},
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CompleteRequest",
			request: JSONRPCRequest{
				Request: &CompleteRequest{
					Request: Request{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "SetLevelRequest",
			request: JSONRPCRequest{
				Request: &SetLevelRequest{
					Request: Request{
						Method: SetLevelRequestMethod,
					},
					Params: SetLevelRequestParams{
						Level: LoggingLevelEmergency,
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "GetPromptRequest",
			request: JSONRPCRequest{
				Request: &GetPromptRequest{
					Request: Request{
						Method: GetPromptRequestMethod,
					},
					Params: GetPromptRequestParams{
						Name: "foo",
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListPromptsRequest",
			request: JSONRPCRequest{
				Request: &ListPromptsRequest{
					Request: Request{
						Method: ListPromptsRequestMethod,
					},
					Params: &PaginatedRequestParams{
						Cursor: &[]Cursor{"test-cursor"}[0],
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListResourcesRequest",
			request: JSONRPCRequest{
				Request: &ListResourcesRequest{
					Request: Request{
						Method: ListResourcesRequestMethod,
					},
					Params: &PaginatedRequestParams{
						Cursor: &[]Cursor{"test-cursor"}[0],
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListResourceTemplatesRequest",
			request: JSONRPCRequest{
				Request: &ListResourceTemplatesRequest{
					Request: Request{
						Method: ListResourceTemplatesRequestMethod,
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListRootsRequest",
			request: JSONRPCRequest{
				Request: &ListRootsRequest{
					Request: Request{
						Method: ListRootsRequestMethod,
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ReadResourceRequest",
			request: JSONRPCRequest{
				Request: &ReadResourceRequest{
					Request: Request{
						Method: ReadResourceRequestMethod,
					},
					Params: &ReadResourceRequestParams{
						URI: "foo://bar",
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "SubscribeRequest",
			request: JSONRPCRequest{
				Request: &SubscribeRequest{
					Request: Request{
						Method: SubscribeRequestMethod,
					},
					Params: SubscribeRequestParams{
						URI: "foo://bar",
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "UnsubscribeRequest",
			request: JSONRPCRequest{
				Request: &UnsubscribeRequest{
					Request: Request{
						Method: SubscribeRequestMethod,
					},
					Params: UnsubscribeRequestParams{
						URI: "foo://bar",
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CallToolRequest",
			request: JSONRPCRequest{
				Request: &CallToolRequest{
					Request: Request{
						Method: CallToolRequestMethod,
					},
					Params: CallToolRequestParams{
						Name: "CallToolName",
						Arguments: CallToolRequestParamsArguments{
							"arg1": "one",
						},
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CreateMessageRequest",
			request: JSONRPCRequest{
				Request: &CreateMessageRequest{
					Request: Request{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Marshal %s failed: %v", tt.name, err)
			}

			// Test unmarshaling
			var decoded JSONRPCRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal %s failed: %v", tt.name, err)
			}

			// Test that the decoded request matches the original
			reencoded, err := json.Marshal(decoded)
			if err != nil {
				t.Fatalf("Re-marshal of decoded %s failed: %v", tt.name, err)
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
		notification JSONRPCNotification
	}{
		{
			name: "ProgressNotification",
			notification: JSONRPCNotification{
				Notification: &ProgressNotification{
					Notification: Notification{
						Method: ProgressNotificationMethod,
					},
					Params: ProgressNotificationParams{
						Progress:      0.6,
						ProgressToken: ProgressToken{Value: uint64(100)},
					},
				},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "InitializedNotification",
			notification: JSONRPCNotification{
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
			notification: JSONRPCNotification{
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
			notification: JSONRPCNotification{
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
			notification: JSONRPCNotification{
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
			notification: JSONRPCNotification{
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
			notification: JSONRPCNotification{
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
			notification: JSONRPCNotification{
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
			notification: JSONRPCNotification{
				Notification: &CancelledNotification{
					Notification: Notification{
						Method: CancelledNotificationMethod,
					},
					Params: CancelledNotificationParams{
						Reason:    &[]string{"some reason"}[0],
						RequestID: RequestID{Value: uint64(3)},
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
				t.Fatalf("Marshal %s failed: %v", tt.name, err)
			}

			// Test unmarshaling
			var decoded JSONRPCNotification
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal %s failed: %v", tt.name, err)
			}

			// Test that the decoded notification matches the original
			reencoded, err := json.Marshal(decoded)
			if err != nil {
				t.Fatalf("Re-marshal of decoded %s failed: %v", tt.name, err)
			}

			// Unmarshal both original and re-encoded JSON into maps
			var originalMap, reencodedMap map[string]any
			if err := json.Unmarshal(data, &originalMap); err != nil {
				t.Fatalf("Failed to unmarshal original JSON: %v", err)
			}
			if err := json.Unmarshal(reencoded, &reencodedMap); err != nil {
				t.Fatalf("Failed to unmarshal re-encoded JSON: %v", err)
			}

			// Compare the maps
			if !reflect.DeepEqual(originalMap, reencodedMap) {
				t.Errorf("Re-encoded %s doesn't match original.\nOriginal: %s\nRe-encoded: %s",
					tt.name, string(data), string(reencoded))
			}
		})
	}
}

func TestResponseMessages(t *testing.T) {
	tests := []struct {
		name     string
		response JSONRPCResponse
	}{
		{
			name: "PingResult",
			response: JSONRPCResponse{
				Result: &PingResult{
					Result: Result{
						Meta: ResultMeta{
							"msg": "pingResult",
						},
					},
				},
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CreateMessageResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListRootsResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CompleteResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "InitializeResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "GetPromptResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListPromptsResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListResourcesResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListResourceTemplatesResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ReadResourceResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "CallToolResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
		{
			name: "ListToolsResult",
			response: JSONRPCResponse{
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
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Marshal %s failed: %v", tt.name, err)
			}

			// Test unmarshaling
			var decoded JSONRPCResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal %s failed: %v", tt.name, err)
			}

			// Test that the decoded response matches the original
			reencoded, err := json.Marshal(decoded)
			if err != nil {
				t.Fatalf("Re-marshal of decoded %s failed: %v", tt.name, err)
			}

			// Unmarshal both original and re-encoded JSON into maps
			var originalMap, reencodedMap map[string]any
			if err := json.Unmarshal(data, &originalMap); err != nil {
				t.Fatalf("Failed to unmarshal original JSON: %v", err)
			}
			if err := json.Unmarshal(reencoded, &reencodedMap); err != nil {
				t.Fatalf("Failed to unmarshal re-encoded JSON: %v", err)
			}

			// Compare the maps
			if !reflect.DeepEqual(originalMap, reencodedMap) {
				t.Errorf("Re-encoded %s doesn't match original.\nOriginal: %s\nRe-encoded: %s",
					tt.name, string(data), string(reencoded))
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name  string
		error JSONRPCError
	}{
		{
			name: "MethodNotFoundError",
			error: JSONRPCError{
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
				Err: Error{
					Code:    JSONRPCMethodNotFoundError,
					Message: "Method not found",
				},
			},
		},
		{
			name: "InvalidParamsError",
			error: JSONRPCError{
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
				Err: Error{
					Code:    JSONRPCInvalidParamError,
					Message: "Invalid parameters",
				},
			},
		},
		{
			name: "InternalError",
			error: JSONRPCError{
				ID:      RequestID{Value: uint64(2)},
				Version: JSONRPCVersion,
				Err: Error{
					Code:    JSONRPCInternalError,
					Message: "Internal error occurred",
				},
			},
		},
		{
			name: "CustomError",
			error: JSONRPCError{
				ID:      RequestID{Value: uint64(2)},
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
				t.Fatalf("Marshal %s failed: %v", tt.name, err)
			}

			// Test unmarshaling
			var decoded JSONRPCError
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal %s failed: %v", tt.name, err)
			}

			// Test that the decoded error matches the original
			reencoded, err := json.Marshal(decoded)
			if err != nil {
				t.Fatalf("Re-marshal of decoded %s failed: %v", tt.name, err)
			}

			// Unmarshal both original and re-encoded JSON into maps
			var originalMap, reencodedMap map[string]any
			if err := json.Unmarshal(data, &originalMap); err != nil {
				t.Fatalf("Failed to unmarshal original JSON: %v", err)
			}
			if err := json.Unmarshal(reencoded, &reencodedMap); err != nil {
				t.Fatalf("Failed to unmarshal re-encoded JSON: %v", err)
			}

			// Compare the maps
			if !reflect.DeepEqual(originalMap, reencodedMap) {
				t.Errorf("Re-encoded %s doesn't match original.\nOriginal: %s\nRe-encoded: %s",
					tt.name, string(data), string(reencoded))
			}
		})
	}
}
