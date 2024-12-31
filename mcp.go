package mcp

import (
	"encoding/json"
	"fmt"
)

const (
	JSONRPCVersion = "2.0"
	LatestVersion  = "2024-11-05"
)

var supportedVersions = map[string]struct{}{
	LatestVersion: {},
	"2024-10-07":  {},
}

func IsSupportedVersion(version string) bool {
	_, ok := supportedVersions[version]
	return ok
}

// RequestT must be implemented by all Requests.
type RequestT[T ID] interface {
	json.Unmarshaler
	GetMethod() RequestMethod
	isRequest()
}

// NotificationT must be implemented by all Notifications.
type NotificationT[T ID] interface {
	json.Unmarshaler
	GetMethod() RequestMethod
	isNotification()
}

// ResultT must be implemented by all Results.
type ResultT interface {
	isResult()
}

// JSONRPCMessage sent over transport
// JSONRPCRequest | JSONRPCResponse | JSONRPCNotification | JSONRPCError
type JSONRPCMessage interface {
	JSONRPCMessageType() JSONRPCMessageType
}

// JSONRPCMessageType identifies JSONRPC message.
type JSONRPCMessageType string

const (
	JSONRPCRequestMsgType      JSONRPCMessageType = "request"
	JSONRPCNotificationMsgType JSONRPCMessageType = "notification"
	JSONRPCResponseMsgType     JSONRPCMessageType = "response"
	JSONRPCErrorMsgType        JSONRPCMessageType = "error"
)

// JSONRPCRequest as defined in the MCP schema.
type JSONRPCRequest[T ID] struct {
	// Request is a specific RPC request.
	Request RequestT[T] `json:"-"`
	// ID corresponds to the JSON schema field "id".
	ID RequestID[T] `json:"id"`
	// Version corresponds to the JSON RPC Versiom.
	// It must be set to JSONRPCVersion
	Version string `json:"jsonrpc"`
}

// Implement JSONRPCMessageTyper
func (j JSONRPCRequest[T]) JSONRPCMessageType() JSONRPCMessageType {
	return JSONRPCRequestMsgType
}

// MarshalJSON implements json.Marshaler.
func (j JSONRPCRequest[T]) MarshalJSON() ([]byte, error) {
	switch r := j.Request.(type) {
	case *PingRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*PingRequest[T]
		}{
			ID:          j.ID.Value,
			Version:     j.Version,
			PingRequest: r,
		})
	case *InitializeRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*InitializeRequest[T]
		}{
			ID:                j.ID.Value,
			Version:           j.Version,
			InitializeRequest: r,
		})
	case *CompleteRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*CompleteRequest[T]
		}{
			ID:              j.ID.Value,
			Version:         j.Version,
			CompleteRequest: r,
		})
	case *SetLevelRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*SetLevelRequest[T]
		}{
			ID:              j.ID.Value,
			Version:         j.Version,
			SetLevelRequest: r,
		})
	case *GetPromptRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*GetPromptRequest[T]
		}{
			ID:               j.ID.Value,
			Version:          j.Version,
			GetPromptRequest: r,
		})
	case *ListPromptsRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*ListPromptsRequest[T]
		}{
			ID:                 j.ID.Value,
			Version:            j.Version,
			ListPromptsRequest: r,
		})
	case *ListResourcesRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*ListResourcesRequest[T]
		}{
			ID:                   j.ID.Value,
			Version:              j.Version,
			ListResourcesRequest: r,
		})
	case *ListResourceTemplatesRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*ListResourceTemplatesRequest[T]
		}{
			ID:                           j.ID.Value,
			Version:                      j.Version,
			ListResourceTemplatesRequest: r,
		})
	case *ListToolsRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*ListToolsRequest[T]
		}{
			ID:               j.ID.Value,
			Version:          j.Version,
			ListToolsRequest: r,
		})
	case *ReadResourceRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*ReadResourceRequest[T]
		}{
			ID:                  j.ID.Value,
			Version:             j.Version,
			ReadResourceRequest: r,
		})
	case *SubscribeRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*SubscribeRequest[T]
		}{
			ID:               j.ID.Value,
			Version:          j.Version,
			SubscribeRequest: r,
		})
	case *UnsubscribeRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*UnsubscribeRequest[T]
		}{
			ID:                 j.ID.Value,
			Version:            j.Version,
			UnsubscribeRequest: r,
		})
	case *CallToolRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*CallToolRequest[T]
		}{
			ID:              j.ID.Value,
			Version:         j.Version,
			CallToolRequest: r,
		})
	case *CreateMessageRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*CreateMessageRequest[T]
		}{
			ID:                   j.ID.Value,
			Version:              j.Version,
			CreateMessageRequest: r,
		})
	case *ListRootsRequest[T]:
		return json.Marshal(struct {
			ID      T      `json:"id"`
			Version string `json:"jsonrpc"`
			*ListRootsRequest[T]
		}{
			ID:               j.ID.Value,
			Version:          j.Version,
			ListRootsRequest: r,
		})
	}
	return nil, fmt.Errorf("unknown request type: %T", j.Request)
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONRPCRequest[T]) UnmarshalJSON(b []byte) error {
	var req struct {
		ID      json.RawMessage `json:"id"`
		Method  RequestMethod   `json:"method"`
		Version string          `json:"jsonrpc"`
	}
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	if len(req.ID) == 0 {
		return fmt.Errorf("field 'id' in JSONRPCRequest: required")
	}
	if req.Version != JSONRPCVersion {
		return fmt.Errorf("invalid 'jsonrpc' in JSONRPCRequest: %q", req.Version)
	}
	if _, ok := enumRequestMethod[req.Method]; !ok {
		return fmt.Errorf("invalid request method in JSONRPCRequest: %q", req.Method)
	}

	// parse the ID
	var requestID RequestID[T]
	if err := json.Unmarshal(req.ID, &requestID); err != nil {
		return fmt.Errorf("failed to unmarshal 'id': %w", err)
	}
	j.ID = requestID
	j.Version = req.Version

	switch req.Method {
	case PingRequestMethod:
		var pr PingRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case InitializeRequestMethod:
		var ir InitializeRequest[T]
		if err := json.Unmarshal(b, &ir); err != nil {
			return err
		}
		j.Request = &ir
	case CompleteRequestMethod:
		var cr CompleteRequest[T]
		if err := json.Unmarshal(b, &cr); err != nil {
			return err
		}
		j.Request = &cr
	case SetLevelRequestMethod:
		var cr SetLevelRequest[T]
		if err := json.Unmarshal(b, &cr); err != nil {
			return err
		}
		j.Request = &cr
	case GetPromptRequestMethod:
		var pr GetPromptRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case ListPromptsRequestMethod:
		var pr ListPromptsRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case ListResourcesRequestMethod:
		var pr ListResourcesRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case ListResourceTemplatesRequestMethod:
		var pr ListResourceTemplatesRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case ListToolsRequestMethod:
		var pr ListToolsRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case ReadResourceRequestMethod:
		var pr ReadResourceRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case SubscribeRequestMethod:
		var pr SubscribeRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case UnsubscribeRequestMethod:
		var pr UnsubscribeRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case CallToolRequestMethod:
		var pr CallToolRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case CreateMessageRequestMethod:
		var pr CreateMessageRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	case ListRootsRequestMethod:
		var pr ListRootsRequest[T]
		if err := json.Unmarshal(b, &pr); err != nil {
			return err
		}
		j.Request = &pr
	default:
		return fmt.Errorf("unsupported request: %q", req.Method)
	}
	return nil
}

// A notification which does not expect a response.
type JSONRPCNotification[T ID] struct {
	// Notification is a specific RPC notification.
	Notification NotificationT[T] `json:"-"`
	// Version corresponds to the JSON schema field "jsonrpc".
	Version string `json:"jsonrpc"`
}

// Implement JSONRPCMessageTyper
func (j JSONRPCNotification[T]) JSONRPCMessageType() JSONRPCMessageType {
	return JSONRPCNotificationMsgType
}

// MarshalJSON implements json.Marshaler.
func (j JSONRPCNotification[T]) MarshalJSON() ([]byte, error) {
	switch n := j.Notification.(type) {
	case *ProgressNotification[T]:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*ProgressNotification[T]
		}{
			Version:              j.Version,
			ProgressNotification: n,
		})
	case *InitializedNotification:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*InitializedNotification
		}{
			Version:                 j.Version,
			InitializedNotification: n,
		})
	case *RootsListChangedNotification:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*RootsListChangedNotification
		}{
			Version:                      j.Version,
			RootsListChangedNotification: n,
		})
	case *LoggingMessageNotification:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*LoggingMessageNotification
		}{
			Version:                    j.Version,
			LoggingMessageNotification: n,
		})
	case *ResourceUpdatedNotification:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*ResourceUpdatedNotification
		}{
			Version:                     j.Version,
			ResourceUpdatedNotification: n,
		})
	case *ResourceListChangedNotification:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*ResourceListChangedNotification
		}{
			Version:                         j.Version,
			ResourceListChangedNotification: n,
		})
	case *ToolListChangedNotification:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*ToolListChangedNotification
		}{
			Version:                     j.Version,
			ToolListChangedNotification: n,
		})
	case *PromptListChangedNotification:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*PromptListChangedNotification
		}{
			Version:                       j.Version,
			PromptListChangedNotification: n,
		})
	case *CancelledNotification[T]:
		return json.Marshal(struct {
			Version string `json:"jsonrpc"`
			*CancelledNotification[T]
		}{
			Version:               j.Version,
			CancelledNotification: n,
		})
	}
	return nil, fmt.Errorf("unknown notification type: %T", j.Notification)
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONRPCNotification[T]) UnmarshalJSON(b []byte) error {
	var n struct {
		Version string        `json:"jsonrpc"`
		Method  RequestMethod `json:"method"`
	}
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	if n.Version != JSONRPCVersion {
		return fmt.Errorf("invalid 'jsonrpc' in JSONRPCNotification: %q", n.Version)
	}
	if _, ok := enumRequestMethod[n.Method]; !ok {
		return fmt.Errorf("invalid method in JSONRPCNotification: %q", n.Method)
	}

	j.Version = n.Version

	switch n.Method {
	case ProgressNotificationMethod:
		var pn ProgressNotification[T]
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	case InitializedNotificationMethod:
		var pn InitializedNotification
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	case RootsListChangedNotificationMethod:
		var pn RootsListChangedNotification
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	case LoggingMessageNotificationMethod:
		var pn LoggingMessageNotification
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	case ResourceUpdatedNotificationMethod:
		var pn ResourceUpdatedNotification
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	case ResourceListChangedNotificationMethod:
		var pn ResourceListChangedNotification
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	case ToolListChangedNotificationMethod:
		var pn ToolListChangedNotification
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	case PromptListChangedNotificationMethod:
		var pn PromptListChangedNotification
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	case CancelledNotificationMethod:
		var pn CancelledNotification[T]
		if err := json.Unmarshal(b, &pn); err != nil {
			return err
		}
		j.Notification = &pn
	default:
		return fmt.Errorf("unsupported notification: %q", n.Method)
	}
	return nil
}

// A successful (non-error) response to a request.
type JSONRPCResponse[T ID] struct {
	// Result for a specific RPC Result.
	Result ResultT `json:"result"`
	// ID corresponds to the JSON schema field "id".
	ID RequestID[T] `json:"id"`
	// Version corresponds to the JSON schema field "jsonrpc".
	Version string `json:"jsonrpc"`
}

// Implement JSONRPCMessageTyper
func (j JSONRPCResponse[T]) JSONRPCMessageType() JSONRPCMessageType {
	return JSONRPCResponseMsgType
}

// UnmarshalJSON implements json.Unmarshaler.
// NOTE: Results do not have any differentiating field
// which makes unmarshaling in Go comically painful.
// We might want to switch to `any` and lose type safety.
func (j *JSONRPCResponse[T]) UnmarshalJSON(b []byte) error {
	var resp struct {
		Result  json.RawMessage `json:"result"`
		ID      json.RawMessage `json:"id"`
		Version string          `json:"jsonrpc"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return err
	}
	if len(resp.ID) == 0 {
		return fmt.Errorf("field 'id' in JSONRPCRequest: required")
	}
	if len(resp.Result) == 0 {
		return fmt.Errorf("field result in JSONRPCResponse: required")
	}
	if resp.Version != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc version JSONRPCResponse: %q", resp.Version)
	}

	// parse the ID
	var respID RequestID[T]
	if err := json.Unmarshal(resp.ID, &respID); err != nil {
		return fmt.Errorf("failed to unmarshal 'id': %w", err)
	}
	j.ID = respID
	j.Version = resp.Version

	// Try unmarshaling into each possible Result type

	// Try CreateMessageResult
	var msgResult CreateMessageResult
	if err := json.Unmarshal(resp.Result, &msgResult); err == nil {
		j.Result = &msgResult
		return nil
	}

	// Try ListRootsResult
	var listRootsResult ListRootsResult
	if err := json.Unmarshal(resp.Result, &listRootsResult); err == nil {
		j.Result = &listRootsResult
		return nil
	}

	// Try InitializeResult
	var initResult InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err == nil {
		j.Result = &initResult
		return nil
	}

	// Try CompleteResult
	var compResult CompleteResult
	if err := json.Unmarshal(resp.Result, &compResult); err == nil {
		j.Result = &compResult
		return nil
	}

	// Try GetPromptResult
	var getPrResult GetPromptResult
	if err := json.Unmarshal(resp.Result, &getPrResult); err == nil {
		j.Result = &getPrResult
		return nil
	}

	// Try ListPromptsResult
	var listPrResult ListPromptsResult
	if err := json.Unmarshal(resp.Result, &listPrResult); err == nil {
		j.Result = &listPrResult
		return nil
	}

	// Try ListResourcesResult
	var listResResult ListResourcesResult
	if err := json.Unmarshal(resp.Result, &listResResult); err == nil {
		j.Result = &listResResult
		return nil
	}

	// Try ListResourceTemplatesResult
	var listTemplRes ListResourceTemplatesResult
	if err := json.Unmarshal(resp.Result, &listTemplRes); err == nil {
		j.Result = &listTemplRes
		return nil
	}

	// Try ReadResourceResult
	var readResResult ReadResourceResult
	if err := json.Unmarshal(resp.Result, &readResResult); err == nil {
		j.Result = &readResResult
		return nil
	}

	// Try CallToolResult
	var callToolRes CallToolResult
	if err := json.Unmarshal(resp.Result, &callToolRes); err == nil {
		j.Result = &callToolRes
		return nil
	}

	// Try ListToolsResult
	var listToolsRes ListToolsResult
	if err := json.Unmarshal(resp.Result, &listToolsRes); err == nil {
		j.Result = &listToolsRes
		return nil
	}

	return fmt.Errorf("unsupported result")
}

type JSONRPCErrorCode int

const (
	JSONRPCParseError          JSONRPCErrorCode = -32700
	JSONRPCInvalidRequestError JSONRPCErrorCode = -32600
	JSONRPCMethodNotFoundError JSONRPCErrorCode = -32601
	JSONRPCInvalidParamError   JSONRPCErrorCode = -32602
	JSONRPCInternalError       JSONRPCErrorCode = -32603
	JSONRPCConnectionClosed    JSONRPCErrorCode = -1
	JSONRPCRequestTimeout      JSONRPCErrorCode = -2
)

type Error struct {
	// The error type that occurred.
	Code JSONRPCErrorCode `json:"code"`
	// A short description of the error. The message SHOULD be limited to a concise
	// single sentence.
	Message string `json:"message"`
	// Additional information about the error. The value of this member is defined by
	// the sender (e.g. detailed error information, nested errors etc.).
	Data any `json:"data,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Error) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["code"]; raw != nil && !ok {
		return fmt.Errorf("field code in JSONRPCErrorError: required")
	}
	if _, ok := raw["message"]; raw != nil && !ok {
		return fmt.Errorf("field message in JSONRPCErrorError: required")
	}
	var e Error
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}
	*j = e
	return nil
}

func (j Error) Error() string {
	return fmt.Sprintf("error %d: %s", j.Code, j.Message)
}

// A response to a request that indicates an error occurred.
type JSONRPCError[T ID] struct {
	// ID corresponds to the JSON schema field "id".
	ID RequestID[T] `json:"id"`
	// Version corresponds to the JSON schema field "jsonrpc".
	Version string `json:"jsonrpc"`
	// Err corresponds to the JSON schema field "error".
	Err Error `json:"error"`
}

// Implement JSONRPCMessageTyper
func (j JSONRPCError[T]) JSONRPCMessageType() JSONRPCMessageType {
	return JSONRPCErrorMsgType
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSONRPCError[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["error"]; raw != nil && !ok {
		return fmt.Errorf("field error in JSONRPCError: required")
	}
	if _, ok := raw["id"]; raw != nil && !ok {
		return fmt.Errorf("field id in JSONRPCError: required")
	}
	val, ok := raw["jsonrpc"]
	if raw != nil && !ok {
		return fmt.Errorf("field jsonrpc in JSONRPCRequest: required")
	}
	if strVal, ok := val.(string); !ok || strVal != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc in JSONRPCNotification: %v", val)
	}
	var e JSONRPCError[T]
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}
	*j = e
	return nil
}

func (j *JSONRPCError[T]) Error() string {
	return fmt.Sprintf("ID: %v, Error: %v", j.ID, j.Err)
}

/////////////////
/// REQUESTS  ///
/////////////////

// ID is used as a generic constraint.
type ID interface {
	~uint64
}

// RequestID is a uniquely identifying ID for a request in JSON-RPC.
// NOTE: RequestID is defined in the spec as string | number
// But Go doesn't have sum types and working around them requires
// jumping through retarded hoops so we're sticking with ~uint64 for now.
type RequestID[T ID] struct {
	Value T `json:"-"`
}

// MarshalJSON marshals RequestID.
func (r RequestID[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Value)
}

// UnmarshalJSON deserializes the RequestID from JSON.
func (r *RequestID[T]) UnmarshalJSON(data []byte) error {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("failed to unmarshal RequestID: %w", err)
	}
	r.Value = v
	return nil
}

// A progress token, used to associate progress
// NOTE: ProgressToken is defined in the spec as string | number
// But Go type system is very sad, so we are sticking with uint64 for now.
type ProgressToken[T ID] struct {
	Value T `json:"-"`
}

// MarshalJSON serializes the ProgressToken to JSON.
func (p ProgressToken[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Value)
}

// UnmarshalJSON deserializes the ProgressToken from JSON.
func (p *ProgressToken[T]) UnmarshalJSON(data []byte) error {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("failed to unmarshal ProgressToken: %w", err)
	}
	p.Value = v
	return nil
}

type RequestMethod string

const (
	PingRequestMethod                     RequestMethod = "ping"
	InitializeRequestMethod               RequestMethod = "initialize"
	CompleteRequestMethod                 RequestMethod = "completion/complete"
	SetLevelRequestMethod                 RequestMethod = "logging/setLevel"
	ResourceListChangedNotificationMethod RequestMethod = "notifications/resources/list_changed"
	InitializedNotificationMethod         RequestMethod = "notifications/initialized"
	ProgressNotificationMethod            RequestMethod = "notifications/progress"
	CancelledNotificationMethod           RequestMethod = "notifications/cancelled"
	ResourceUpdatedNotificationMethod     RequestMethod = "notifications/resources/updated"
	PromptListChangedNotificationMethod   RequestMethod = "notifications/prompts/list_changed"
	ToolListChangedNotificationMethod     RequestMethod = "notifications/tools/list_changed"
	RootsListChangedNotificationMethod    RequestMethod = "notifications/roots/list_changed"
	LoggingMessageNotificationMethod      RequestMethod = "notifications/message"
	ListPromptsRequestMethod              RequestMethod = "prompts/list"
	GetPromptRequestMethod                RequestMethod = "prompts/get"
	ListResourcesRequestMethod            RequestMethod = "resources/list"
	ListResourceTemplatesRequestMethod    RequestMethod = "resources/templates/list"
	SubscribeRequestMethod                RequestMethod = "resources/subscribe"
	UnsubscribeRequestMethod              RequestMethod = "resources/unsubscribe"
	ReadResourceRequestMethod             RequestMethod = "resources/read"
	ListRootsRequestMethod                RequestMethod = "roots/list"
	CreateMessageRequestMethod            RequestMethod = "sampling/createMessage"
	ListToolsRequestMethod                RequestMethod = "tools/list"
	CallToolRequestMethod                 RequestMethod = "tools/call"
)

var enumRequestMethod = map[RequestMethod]struct{}{
	PingRequestMethod:                     {},
	InitializeRequestMethod:               {},
	CompleteRequestMethod:                 {},
	SetLevelRequestMethod:                 {},
	ResourceListChangedNotificationMethod: {},
	InitializedNotificationMethod:         {},
	ProgressNotificationMethod:            {},
	CancelledNotificationMethod:           {},
	ResourceUpdatedNotificationMethod:     {},
	PromptListChangedNotificationMethod:   {},
	ToolListChangedNotificationMethod:     {},
	RootsListChangedNotificationMethod:    {},
	LoggingMessageNotificationMethod:      {},
	ListPromptsRequestMethod:              {},
	GetPromptRequestMethod:                {},
	ListResourcesRequestMethod:            {},
	ListResourceTemplatesRequestMethod:    {},
	SubscribeRequestMethod:                {},
	UnsubscribeRequestMethod:              {},
	ReadResourceRequestMethod:             {},
	ListRootsRequestMethod:                {},
	CreateMessageRequestMethod:            {},
	ListToolsRequestMethod:                {},
	CallToolRequestMethod:                 {},
}

type RequestParamsMeta[T ID] struct {
	// If specified, the caller is requesting out-of-band progress notifications for
	// this request (as represented by notifications/progress). The value of this
	// parameter is an opaque token that will be attached to any subsequent
	// notifications. The receiver is not obligated to provide these notifications.
	ProgressToken *ProgressToken[T] `json:"progressToken,omitempty"`
}

type RequestParams[T ID] struct {
	// Meta corresponds to the JSON schema field "_meta".
	Meta *RequestParamsMeta[T] `json:"_meta,omitempty"`
	// AdditionalProperties are reserved for future use.
	AdditionalProperties any `json:",omitempty"`
}

type Request[T ID] struct {
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params *RequestParams[T] `json:"params,omitempty"`
}

// An opaque token used to represent a cursor for pagination.
type Cursor string

type PaginatedRequestParams struct {
	// An opaque token representing the current pagination position.
	// If provided, the server should return results starting after this cursor.
	Cursor *Cursor `json:"cursor,omitempty"`
}

// NOTE: this is unused due to its causing panics when a request embeds this.
// TODO: investigate panics that happen when a *Request embeds this type!
type PaginatedRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params *PaginatedRequestParams `json:"params,omitempty"`
}

type PingRequestParams[T ID] struct {
	// Meta corresponds to the JSON schema field "_meta".
	Meta *PingRequestParamsMeta[T] `json:"_meta,omitempty"`
	// AdditionalProperties reserved for future use.
	AdditionalProperties map[string]any `json:",omitempty"`
}

type PingRequestParamsMeta[T ID] struct {
	// If specified, the caller is requesting out-of-band progress notifications for
	// this request (as represented by notifications/progress). The value of this
	// parameter is an opaque token that will be attached to any subsequent
	// notifications. The receiver is not obligated to provide these notifications.
	ProgressToken *ProgressToken[T] `json:"progressToken,omitempty"`
}

// A ping, issued by either the server or the client, to check that the other party
// is still alive. The receiver must promptly respond, or else may be disconnected.
type PingRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params *PingRequestParams[T] `json:"params,omitempty"`
}

func (p *PingRequest[T]) GetMethod() RequestMethod {
	return p.Request.Method
}

func (p *PingRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (p *PingRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in PingRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != PingRequestMethod {
		return fmt.Errorf("invalid field method in PingRequest: %v", strVal)
	}
	type pingAlias PingRequest[T]
	var req pingAlias
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*p = PingRequest[T](req)
	return nil
}

type ClientCapabilitiesExperimental map[string]any

// Present if the client supports listing roots.
type ClientCapabilitiesRoots struct {
	// Whether the client supports notifications for changes to the roots list.
	ListChanged *bool `json:"listChanged,omitempty"`
}

// Present if the client supports sampling from an LLM.
type ClientCapabilitiesSampling map[string]any

// Capabilities a client may support. Known capabilities are defined here, in this
// schema, but this is not a closed set: any client can define its own, additional
// capabilities.
type ClientCapabilities struct {
	// Experimental, non-standard capabilities that the client supports.
	Experimental ClientCapabilitiesExperimental `json:"experimental,omitempty"`
	// Present if the client supports listing roots.
	Roots *ClientCapabilitiesRoots `json:"roots,omitempty"`
	// Present if the client supports sampling from an LLM.
	Sampling ClientCapabilitiesSampling `json:"sampling,omitempty"`
}

type InitializeRequestParams struct {
	// Capabilities corresponds to the JSON schema field "capabilities".
	Capabilities ClientCapabilities `json:"capabilities"`
	// ClientInfo corresponds to the JSON schema field "clientInfo".
	ClientInfo Implementation `json:"clientInfo"`
	// The latest version of the Model Context Protocol that the client supports. The
	// client MAY decide to support older versions as well.
	ProtocolVersion string `json:"protocolVersion"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InitializeRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["capabilities"]; raw != nil && !ok {
		return fmt.Errorf("field capabilities in InitializeRequestParams: required")
	}
	if _, ok := raw["clientInfo"]; raw != nil && !ok {
		return fmt.Errorf("field clientInfo in InitializeRequestParams: required")
	}
	if _, ok := raw["protocolVersion"]; raw != nil && !ok {
		return fmt.Errorf("field protocolVersion in InitializeRequestParams: required")
	}
	type aliasInitializeRequestParams InitializeRequestParams
	var params aliasInitializeRequestParams
	if err := json.Unmarshal(b, &params); err != nil {
		return err
	}
	*j = InitializeRequestParams(params)
	return nil
}

// This request is sent from the client to the server when it first connects,
// asking it to begin initialization.
type InitializeRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params InitializeRequestParams `json:"params"`
}

func (j *InitializeRequest[T]) GetMethod() RequestMethod {
	return j.Request.Method
}

func (j *InitializeRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InitializeRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in InitializeRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != InitializeRequestMethod {
		return fmt.Errorf("invalid field method in InitializeRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in InitializeRequest: required")
	}
	type aliasInitializeRequest InitializeRequest[T]
	var req aliasInitializeRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*j = InitializeRequest[T](req)
	return nil
}

// The argument's information
type CompleteRequestParamsArgument struct {
	// The name of the argument
	Name string `json:"name"`
	// The value of the argument to use for completion matching.
	Value string `json:"value"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CompleteRequestParamsArgument) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in CompleteRequestParamsArgument: required")
	}
	if _, ok := raw["value"]; raw != nil && !ok {
		return fmt.Errorf("field value in CompleteRequestParamsArgument: required")
	}
	type aliasCompleteRequestParamsArgument CompleteRequestParamsArgument
	var c aliasCompleteRequestParamsArgument
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}
	*j = CompleteRequestParamsArgument(c)
	return nil
}

type ReferenceType string

const (
	PromptReferenceType   ReferenceType = "ref/prompt"
	ResourceReferenceType ReferenceType = "ref/resource"
)

// NOTE: Go doesn't have sum types so this is a workaround for
// PromptReference | ResourceReference
type CompleteRequestParamsRef struct {
	PromptReference   *PromptReference
	ResourceReference *ResourceReference
}

// MarshalJSON implements custom JSON marshaling
func (c CompleteRequestParamsRef) MarshalJSON() ([]byte, error) {
	if c.PromptReference != nil {
		if c.PromptReference.Type != PromptReferenceType {
			return nil, fmt.Errorf("invalid ref type for PromptReference: %q", c.PromptReference.Type)
		}
		return json.Marshal(c.PromptReference)
	}
	if c.ResourceReference != nil {
		if c.ResourceReference.Type != ResourceReferenceType {
			return nil, fmt.Errorf("invalid ref type for ResourceReference: %q", c.ResourceReference.Type)
		}
		return json.Marshal(c.ResourceReference)
	}
	return nil, fmt.Errorf("neither PromptReference nor ResourceReference is set")
}

func (c *CompleteRequestParamsRef) UnmarshalJSON(data []byte) error {
	var p PromptReference
	if err := json.Unmarshal(data, &p); err == nil && p.Type == PromptReferenceType {
		c.PromptReference = &p
		return nil
	}
	var r ResourceReference
	if err := json.Unmarshal(data, &r); err == nil && r.Type == ResourceReferenceType {
		c.ResourceReference = &r
		return nil
	}
	return fmt.Errorf("invalid CompleteRequestParamsRef type: %s", r.Type)
}

type CompleteRequestParams struct {
	// Ref corresponds to the JSON schema field "ref".
	// PromptReference | ResourceReference
	Ref CompleteRequestParamsRef `json:"ref"`
	// The argument's information
	Argument CompleteRequestParamsArgument `json:"argument"`
}

// A request from the client to the server, to ask for completion options.
type CompleteRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params CompleteRequestParams `json:"params"`
}

func (c *CompleteRequest[T]) GetMethod() RequestMethod {
	return c.Request.Method
}

func (c *CompleteRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CompleteRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in CompleteRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != CompleteRequestMethod {
		return fmt.Errorf("invalid field method in CompleteRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in CompleteRequest: required")
	}
	type aliasCompleteRequest CompleteRequest[T]
	var cr aliasCompleteRequest
	if err := json.Unmarshal(b, &cr); err != nil {
		return err
	}
	*c = CompleteRequest[T](cr)
	return nil
}

type LoggingLevel string

const (
	LoggingLevelAlert     LoggingLevel = "alert"
	LoggingLevelCritical  LoggingLevel = "critical"
	LoggingLevelDebug     LoggingLevel = "debug"
	LoggingLevelEmergency LoggingLevel = "emergency"
	LoggingLevelError     LoggingLevel = "error"
	LoggingLevelInfo      LoggingLevel = "info"
	LoggingLevelNotice    LoggingLevel = "notice"
	LoggingLevelWarning   LoggingLevel = "warning"
)

var enumLoggingLevel = map[LoggingLevel]struct{}{
	LoggingLevelAlert:     {},
	LoggingLevelCritical:  {},
	LoggingLevelDebug:     {},
	LoggingLevelEmergency: {},
	LoggingLevelError:     {},
	LoggingLevelInfo:      {},
	LoggingLevelNotice:    {},
	LoggingLevelWarning:   {},
}

type SetLevelRequestParams struct {
	// The level of logging that the client wants to receive from the server. The
	// server should send all logs at this level and higher (i.e., more severe) to the
	// client as notifications/logging/message.
	Level LoggingLevel `json:"level"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SetLevelRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["level"]
	if raw != nil && !ok {
		return fmt.Errorf("field level in SetLevelRequestParams: required")
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid field level in SetLevelRequestParams: %v", strVal)
	}
	if _, valid := enumLoggingLevel[LoggingLevel(strVal)]; !valid {
		return fmt.Errorf("invalid SetLevelRequestParams log level value: %v", strVal)
	}
	type aliasSetLevelRequestParams SetLevelRequestParams
	var req aliasSetLevelRequestParams
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*s = SetLevelRequestParams(req)
	return nil
}

// A request from the client to the server, to enable or adjust logging.
type SetLevelRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params SetLevelRequestParams `json:"params"`
}

func (s *SetLevelRequest[T]) GetMethod() RequestMethod {
	return s.Request.Method
}

func (s *SetLevelRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SetLevelRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in SetLevelRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != SetLevelRequestMethod {
		return fmt.Errorf("invalid field method in SetLevelRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in SetLevelRequest: required")
	}
	type aliasSetLevelRequest SetLevelRequest[T]
	var req aliasSetLevelRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*s = SetLevelRequest[T](req)
	return nil
}

// Arguments to use for templating the prompt.
type GetPromptRequestParamsArguments map[string]string

type GetPromptRequestParams struct {
	// The name of the prompt or prompt template.
	Name string `json:"name"`
	// Arguments to use for templating the prompt.
	Arguments GetPromptRequestParamsArguments `json:"arguments,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (g *GetPromptRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in GetPromptRequestParams: required")
	}
	type aliasGetPromptRequestParams GetPromptRequestParams
	var p aliasGetPromptRequestParams
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*g = GetPromptRequestParams(p)
	return nil
}

// Used by the client to get a prompt provided by the server.
type GetPromptRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params GetPromptRequestParams `json:"params"`
}

func (p *GetPromptRequest[T]) GetMethod() RequestMethod {
	return p.Request.Method
}

func (p *GetPromptRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (p *GetPromptRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in GetPromptRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != GetPromptRequestMethod {
		return fmt.Errorf("invalid field method in GetPromptRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in GetPromptRequest: required")
	}
	type aliasGetPromptRequest GetPromptRequest[T]
	var req aliasGetPromptRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*p = GetPromptRequest[T](req)
	return nil
}

// Sent from the client to request a list of prompts and
// prompt templates the server has.
type ListPromptsRequest[T ID] struct {
	// PaginatedRequest[T]
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params *PaginatedRequestParams `json:"params,omitempty"`
}

func (l *ListPromptsRequest[T]) isRequest() {}

func (l *ListPromptsRequest[T]) GetMethod() RequestMethod {
	return l.Request.Method
}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ListPromptsRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ListPromptsRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ListPromptsRequestMethod {
		return fmt.Errorf("invalid field method in ListPromptsRequest: %v", strVal)
	}
	type aliasListPromptsRequest ListPromptsRequest[T]
	var lp aliasListPromptsRequest
	if err := json.Unmarshal(b, &lp); err != nil {
		return err
	}
	*l = ListPromptsRequest[T](lp)
	return nil
}

// Sent from the client to request a list of resources the server has.
type ListResourcesRequest[T ID] struct {
	Request[T]
	Params *PaginatedRequestParams `json:"params,omitempty"`
}

func (l *ListResourcesRequest[T]) GetMethod() RequestMethod {
	return l.Request.Method
}

func (l *ListResourcesRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ListResourcesRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ListResourcesRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ListResourcesRequestMethod {
		return fmt.Errorf("invalid field method in ListResourcesRequest: %v", strVal)
	}
	type aliasListResourcesRequest ListResourcesRequest[T]
	var req aliasListResourcesRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*l = ListResourcesRequest[T](req)
	return nil
}

// Sent from the client to request a list of resource templates the server has.
type ListResourceTemplatesRequest[T ID] struct {
	Request[T]
	Params *PaginatedRequestParams `json:"params,omitempty"`
}

func (l *ListResourceTemplatesRequest[T]) GetMethod() RequestMethod {
	return l.Request.Method
}

func (l *ListResourceTemplatesRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ListResourceTemplatesRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ListResourceTemplatesRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ListResourceTemplatesRequestMethod {
		return fmt.Errorf("invalid field method in ListResourceTemplatesRequest: %v", strVal)
	}
	type aliasListResourceTemplatesRequest ListResourceTemplatesRequest[T]
	var req aliasListResourceTemplatesRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*l = ListResourceTemplatesRequest[T](req)
	return nil
}

// Sent from the server to request a list of root URIs from the client. Roots allow
// servers to ask for specific directories or files to operate on. A common example
// for roots is providing a set of repositories or directories a server should operate on.
// This request is typically used when the server needs to understand the filesystem
// structure or access specific locations that the client has permission to read from.
type ListRootsRequest[T ID] struct {
	Request[T]
}

func (l *ListRootsRequest[T]) GetMethod() RequestMethod {
	return l.Request.Method
}

func (l *ListRootsRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ListRootsRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ListRootsRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ListRootsRequestMethod {
		return fmt.Errorf("invalid field method in ListRootsRequest: %v", strVal)
	}
	type aliasListRootsRequest ListRootsRequest[T]
	var req aliasListRootsRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*l = ListRootsRequest[T](req)
	return nil
}

// Sent from the client to request a list of tools the server has.
type ListToolsRequest[T ID] struct {
	Request[T]
	Params *PaginatedRequestParams `json:"params,omitempty"`
}

func (l *ListToolsRequest[T]) GetMethod() RequestMethod {
	return l.Request.Method
}

func (l *ListToolsRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ListToolsRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ListToolsRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ListToolsRequestMethod {
		return fmt.Errorf("invalid field method in ListToolsRequest: %v", strVal)
	}
	type aliasListToolsRequest ListToolsRequest[T]
	var req aliasListToolsRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*l = ListToolsRequest[T](req)
	return nil
}

type ReadResourceRequestParams struct {
	// The URI of the resource to read. The URI can use any protocol.
	// It is up to the server how to interpret it.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *ReadResourceRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in ReadResourceRequestParams: required")
	}
	type aliasReadResourceRequestParams ReadResourceRequestParams
	var params aliasReadResourceRequestParams
	if err := json.Unmarshal(b, &params); err != nil {
		return err
	}
	*r = ReadResourceRequestParams(params)
	return nil
}

// Sent from the client to the server, to read a specific resource URI.
type ReadResourceRequest[T ID] struct {
	Request[T]
	// Padams for the request
	Params *ReadResourceRequestParams `json:"params,omitempty"`
}

func (r *ReadResourceRequest[T]) GetMethod() RequestMethod {
	return r.Request.Method
}

func (r *ReadResourceRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (r *ReadResourceRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ReadResourceRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ReadResourceRequestMethod {
		return fmt.Errorf("invalid field method in ReadResourceRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in ReadResourceRequest: required")
	}
	type aliasReadResourceRequest ReadResourceRequest[T]
	var req aliasReadResourceRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*r = ReadResourceRequest[T](req)
	return nil
}

type SubscribeRequestParams struct {
	// The URI of the resource to subscribe to. The URI can use any protocol; it is up
	// to the server how to interpret it.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SubscribeRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in SubscribeRequestParams: required")
	}
	type aliasSubscribeRequestParams SubscribeRequestParams
	var sub aliasSubscribeRequestParams
	if err := json.Unmarshal(b, &sub); err != nil {
		return err
	}
	*j = SubscribeRequestParams(sub)
	return nil
}

// Sent from the client to request resources/updated notifications from the server
// whenever a particular resource changes.
type SubscribeRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params SubscribeRequestParams `json:"params"`
}

func (s *SubscribeRequest[T]) GetMethod() RequestMethod {
	return s.Request.Method
}

func (s *SubscribeRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SubscribeRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in SubscribeRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != SubscribeRequestMethod {
		return fmt.Errorf("invalid field method in SubscribeRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in SubscribeRequest: required")
	}
	type aliasSubscribeRequest SubscribeRequest[T]
	var sub aliasSubscribeRequest
	if err := json.Unmarshal(b, &sub); err != nil {
		return err
	}
	*s = SubscribeRequest[T](sub)
	return nil
}

type UnsubscribeRequestParams struct {
	// The URI of the resource to unsubscribe from.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (u *UnsubscribeRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in UnsubscribeRequestParams: required")
	}
	type aliasUnsubscribeRequestParams UnsubscribeRequestParams
	var params aliasUnsubscribeRequestParams
	if err := json.Unmarshal(b, &params); err != nil {
		return err
	}
	*u = UnsubscribeRequestParams(params)
	return nil
}

// Sent from the client to request cancellation of resources/updated notifications
// from the server. This should follow a previous resources/subscribe request.
type UnsubscribeRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params UnsubscribeRequestParams `json:"params"`
}

func (u *UnsubscribeRequest[T]) GetMethod() RequestMethod {
	return u.Request.Method
}

func (u *UnsubscribeRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (u *UnsubscribeRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in UnsubscribeRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != UnsubscribeRequestMethod {
		return fmt.Errorf("invalid field method in UnsubscribeRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in UnsubscribeRequest: required")
	}
	type aliasUnsubscribeRequest UnsubscribeRequest[T]
	var req aliasUnsubscribeRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}
	*u = UnsubscribeRequest[T](req)
	return nil
}

type CallToolRequestParamsArguments map[string]any

type CallToolRequestParams struct {
	// Arguments corresponds to the JSON schema field "arguments".
	Arguments CallToolRequestParamsArguments `json:"arguments,omitempty"`
	// Name corresponds to the JSON schema field "name".
	Name string `json:"name"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CallToolRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in CallToolRequestParams: required")
	}
	type aliasCallToolRequestParams CallToolRequestParams
	var param aliasCallToolRequestParams
	if err := json.Unmarshal(b, &param); err != nil {
		return err
	}
	*c = CallToolRequestParams(param)
	return nil
}

// Used by the client to invoke a tool provided by the server.
type CallToolRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params CallToolRequestParams `json:"params"`
}

func (c *CallToolRequest[T]) GetMethod() RequestMethod {
	return c.Request.Method
}

func (c *CallToolRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CallToolRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in CallToolRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != CallToolRequestMethod {
		return fmt.Errorf("invalid field method in CallToolRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in CallToolRequest: required")
	}
	type aliasCallToolRequest CallToolRequest[T]
	var tool aliasCallToolRequest
	if err := json.Unmarshal(b, &tool); err != nil {
		return err
	}
	*c = CallToolRequest[T](tool)
	return nil
}

// Hints to use for model selection.
// Keys not declared here are currently left unspecified by the spec and
// are up to the client to interpret.
type ModelHint struct {
	// A hint for a model name.
	//
	// The client SHOULD treat this as a substring of a model name; for example:
	//  - `claude-3-5-sonnet` should match `claude-3-5-sonnet-20241022`
	//  - `sonnet` should match `claude-3-5-sonnet-20241022`,
	// `claude-3-sonnet-20240229`, etc.
	//  - `claude` should match any Claude model
	//
	// The client MAY also map the string to a different provider's model name or a
	// different model family, as long as it fills a similar niche; for example:
	//  - `gemini-1.5-flash` could match `claude-3-haiku-20240307`
	Name *string `json:"name,omitempty"`
}

// The server's preferences for model selection, requested of the client during
// sampling.
//
// Because LLMs can vary along multiple dimensions, choosing the "best" model is
// rarely straightforward.  Different models excel in different areas—some are
// faster but less capable, others are more capable but more expensive, and so
// on. This interface allows servers to express their priorities across multiple
// dimensions to help clients make an appropriate selection for their use case.
//
// These preferences are always advisory. The client MAY ignore them. It is also
// up to the client to decide how to interpret these preferences and how to
// balance them against other considerations.
type ModelPreferences struct {
	// Optional hints to use for model selection.
	//
	// If multiple hints are specified, the client MUST evaluate them in order
	// (such that the first match is taken).
	//
	// The client SHOULD prioritize these hints over the numeric priorities, but
	// MAY still use the priorities to select from ambiguous matches.
	Hints []ModelHint `json:"hints,omitempty"`
	// How much to prioritize cost when selecting a model. A value of 0 means cost
	// is not important, while a value of 1 means cost is the most important
	// factor.
	CostPriority *float64 `json:"costPriority,omitempty"`
	// How much to prioritize sampling speed (latency) when selecting a model. A
	// value of 0 means speed is not important, while a value of 1 means speed is
	// the most important factor.
	SpeedPriority *float64 `json:"speedPriority,omitempty"`
	// How much to prioritize intelligence and capabilities when selecting a
	// model. A value of 0 means intelligence is not important, while a value of 1
	// means intelligence is the most important factor.
	IntelligencePriority *float64 `json:"intelligencePriority,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *ModelPreferences) UnmarshalJSON(b []byte) error {
	type aliasModelPrefs ModelPreferences
	var prefs aliasModelPrefs
	if err := json.Unmarshal(b, &prefs); err != nil {
		return err
	}
	if prefs.CostPriority != nil && 1 < *prefs.CostPriority {
		return fmt.Errorf("field %s: must be <= %v", "costPriority", 1)
	}
	if prefs.CostPriority != nil && 0 > *prefs.CostPriority {
		return fmt.Errorf("field %s: must be >= %v", "costPriority", 0)
	}
	if prefs.IntelligencePriority != nil && 1 < *prefs.IntelligencePriority {
		return fmt.Errorf("field %s: must be <= %v", "intelligencePriority", 1)
	}
	if prefs.IntelligencePriority != nil && 0 > *prefs.IntelligencePriority {
		return fmt.Errorf("field %s: must be >= %v", "intelligencePriority", 0)
	}
	if prefs.SpeedPriority != nil && 1 < *prefs.SpeedPriority {
		return fmt.Errorf("field %s: must be <= %v", "speedPriority", 1)
	}
	if prefs.SpeedPriority != nil && 0 > *prefs.SpeedPriority {
		return fmt.Errorf("field %s: must be >= %v", "speedPriority", 0)
	}
	*m = ModelPreferences(prefs)
	return nil
}

type CreateMsgReqParamsInclCtx string

const (
	CreateMsgReqParamsInclCtxNone       CreateMsgReqParamsInclCtx = "none"
	CreateMsgReqParamsInclCtxAllServers CreateMsgReqParamsInclCtx = "allServers"
	CreateMsgReqParamsInclCtxThisServer CreateMsgReqParamsInclCtx = "thisServer"
)

var enumCreateMsgReqParamsInclCtx = map[CreateMsgReqParamsInclCtx]struct{}{
	CreateMsgReqParamsInclCtxNone:       {},
	CreateMsgReqParamsInclCtxAllServers: {},
	CreateMsgReqParamsInclCtxThisServer: {},
}

// NOTE: Go doesn't have sum types so this is a workaround for
// TextContent | ImageContent
type SamplingMessageContent struct {
	TextContent  *TextContent
	ImageContent *ImageContent
}

// MarshalJSON implements custom JSON marshaling
func (s *SamplingMessageContent) MarshalJSON() ([]byte, error) {
	if s.TextContent != nil {
		if s.TextContent.Type != TextContentType {
			return nil, fmt.Errorf("invalid content type for text content: %q", s.TextContent.Type)
		}
		return json.Marshal(s.TextContent)
	}
	if s.ImageContent != nil {
		if s.ImageContent.Type != ImageContentType {
			return nil, fmt.Errorf("invalid content type for image content: %q", s.ImageContent.Type)
		}
		return json.Marshal(s.ImageContent)
	}
	return nil, fmt.Errorf("neither TextContent nor ImageContent is set")
}

func (s *SamplingMessageContent) UnmarshalJSON(data []byte) error {
	var txt TextContent
	if err := json.Unmarshal(data, &txt); err == nil && txt.Type == TextContentType {
		s.TextContent = &txt
		return nil
	}
	var img ImageContent
	if err := json.Unmarshal(data, &img); err == nil && img.Type == ImageContentType {
		s.ImageContent = &img
		return nil
	}
	return fmt.Errorf("sampling content matches neither TextContent nor ImageContent")
}

// Describes a message issued to or received from an LLM API.
type SamplingMessage struct {
	// Role corresponds to the JSON schema field "role".
	Role Role `json:"role"`
	// Content corresponds to the JSON schema field "content".
	// TextContent | ImageContent
	Content SamplingMessageContent `json:"content"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SamplingMessage) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["content"]; raw != nil && !ok {
		return fmt.Errorf("field content in SamplingMessage: required")
	}
	val, ok := raw["role"]
	if !ok {
		return fmt.Errorf("field role in SamplingMessage: required")
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid field role in SamplingMessage: %v", strVal)
	}
	if _, valid := enumRole[Role(strVal)]; !valid {
		return fmt.Errorf("invalid SamplingMessage role value: %v", strVal)
	}
	type aliasSamplingMessage SamplingMessage
	var sm aliasSamplingMessage
	if err := json.Unmarshal(b, &sm); err != nil {
		return err
	}
	*j = SamplingMessage(sm)
	return nil
}

// Optional metadata to pass through to the LLM provider. The format of this
// metadata is provider-specific.
type CreateMessageRequestParamsMetadata map[string]any

type CreateMessageRequestParams struct {
	// Messages corresponds to the JSON schema field "messages".
	Messages []SamplingMessage `json:"messages"`
	// The server's preferences for which model to select. The client MAY ignore these
	// preferences.
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
	// An optional system prompt the server wants to use for sampling. The client MAY
	// modify or omit this prompt.
	SystemPrompt *string `json:"systemPrompt,omitempty"`
	// A request to include context from one or more MCP servers (including the
	// caller), to be attached to the prompt. The client MAY ignore this request.
	IncludeContext *CreateMsgReqParamsInclCtx `json:"includeContext,omitempty"`
	// Temperature corresponds to the JSON schema field "temperature".
	Temperature *float64 `json:"temperature,omitempty"`
	// The maximum number of tokens to sample, as requested by the server. The client
	// MAY choose to sample fewer tokens than requested.
	MaxTokens int `json:"maxTokens"`
	// Optional metadata to pass through to the LLM provider. The format of this
	// metadata is provider-specific.
	Metadata CreateMessageRequestParamsMetadata `json:"metadata,omitempty"`
	// StopSequences corresponds to the JSON schema field "stopSequences".
	StopSequences []string `json:"stopSequences,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CreateMessageRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["maxTokens"]; raw != nil && !ok {
		return fmt.Errorf("field maxTokens in CreateMessageRequestParams: required")
	}
	if _, ok := raw["messages"]; raw != nil && !ok {
		return fmt.Errorf("field messages in CreateMessageRequestParams: required")
	}
	val := raw["includeContext"]
	if val != nil {
		strVal, ok := val.(string)
		if !ok {
			return fmt.Errorf("invalid field includeContext in CreateMessageRequestParams: %v", strVal)
		}
		if _, ok := enumCreateMsgReqParamsInclCtx[CreateMsgReqParamsInclCtx(strVal)]; !ok {
			return fmt.Errorf("invalid CreateMessageRequestParams includeContext value: %v", strVal)
		}
	}
	type aliasCreateMessageRequestParams CreateMessageRequestParams
	var cm aliasCreateMessageRequestParams
	if err := json.Unmarshal(b, &cm); err != nil {
		return err
	}
	*c = CreateMessageRequestParams(cm)
	return nil
}

// A request from the server to sample an LLM via the client. The client has full
// discretion over which model to select. The client should also inform the user
// before beginning sampling, to allow them to inspect the request (human in the
// loop) and decide whether to approve it.
type CreateMessageRequest[T ID] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params CreateMessageRequestParams `json:"params"`
}

func (c *CreateMessageRequest[T]) GetMethod() RequestMethod {
	return c.Request.Method
}

func (c *CreateMessageRequest[T]) isRequest() {}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CreateMessageRequest[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in CreateMessageRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != CreateMessageRequestMethod {
		return fmt.Errorf("invalid field method in CreateMessageRequest: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in CreateMessageRequest: required")
	}
	type aliasCreateMessageRequest CreateMessageRequest[T]
	var cm aliasCreateMessageRequest
	if err := json.Unmarshal(b, &cm); err != nil {
		return err
	}
	*c = CreateMessageRequest[T](cm)
	return nil
}

//////////////////////
/// NOTIFICATIONS  ///
//////////////////////

// This parameter name is reserved by MCP to allow clients and servers to attach
// additional metadata to their notifications.
type NotificationParamsMeta map[string]any

// NotificationParams passed to notifications.
type NotificationParams struct {
	// This parameter name is reserved by MCP to allow clients and servers to attach
	// additional metadata to their notifications.
	Meta NotificationParamsMeta `json:"_meta,omitempty"`
	// AdditionalProperties for future use
	AdditionalProperties any `json:",omitempty"`
}

type Notification struct {
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params *NotificationParams `json:"params,omitempty"`
}

type ProgressNotificationParams[T ID] struct {
	// The progress thus far. This should increase every time progress is made, even
	// if the total is unknown.
	Progress float64 `json:"progress"`
	// The progress token which was given in the initial request, used to associate
	// this notification with the request that is proceeding.
	ProgressToken ProgressToken[T] `json:"progressToken"`
	// Total number of items to process (or total progress required), if known.
	Total *int64 `json:"total,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *ProgressNotificationParams[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["progress"]; raw != nil && !ok {
		return fmt.Errorf("field progress in ProgressNotificationParams: required")
	}
	if _, ok := raw["progressToken"]; raw != nil && !ok {
		return fmt.Errorf("field progressToken in ProgressNotificationParams: required")
	}
	type aliasProgressNotificationParams ProgressNotificationParams[T]
	var params aliasProgressNotificationParams
	if err := json.Unmarshal(b, &params); err != nil {
		return err
	}
	*p = ProgressNotificationParams[T](params)
	return nil
}

// An out-of-band notification used to inform the receiver of a progress update for
// a long-running request.
type ProgressNotification[T ID] struct {
	Notification
	// Params corresponds to the JSON schema field "params".
	Params ProgressNotificationParams[T] `json:"params"`
}

func (p *ProgressNotification[T]) GetMethod() RequestMethod {
	return p.Notification.Method
}

func (p *ProgressNotification[T]) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (p *ProgressNotification[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ProgressNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ProgressNotificationMethod {
		return fmt.Errorf("invalid field method in ProgressNotification: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in ProgressNotification: required")
	}
	type aliasProgressNotification ProgressNotification[T]
	var pn aliasProgressNotification
	if err := json.Unmarshal(b, &pn); err != nil {
		return err
	}
	*p = ProgressNotification[T](pn)
	return nil
}

// This parameter name is reserved by MCP to allow clients and servers to attach
// additional metadata to their notifications.
type InitializedNotificationParamsMeta map[string]any

type InitializedNotificationParams struct {
	// This parameter name is reserved by MCP to allow clients and servers to attach
	// additional metadata to their notifications.
	Meta InitializedNotificationParamsMeta `json:"_meta,omitempty"`
	// AdditionalProperties reserved for future use.
	AdditionalProperties any `json:",omitempty"`
}

// This notification is sent from the client to the server after initialization has
// finished.
type InitializedNotification struct {
	Notification
	// Params corresponds to the JSON schema field "params".
	Params *InitializedNotificationParams `json:"params,omitempty"`
}

func (j *InitializedNotification) GetMethod() RequestMethod {
	return j.Notification.Method
}

func (j *InitializedNotification) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InitializedNotification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in InitializedNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != InitializedNotificationMethod {
		return fmt.Errorf("invalid field method in InitializedNotification: %v", strVal)
	}
	type aliasInitializedNotification InitializedNotification
	var n aliasInitializedNotification
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*j = InitializedNotification(n)
	return nil
}

// A notification from the client to the server, informing it that the list of
// roots has changed. This notification should be sent whenever the client adds,
// removes, or modifies any root. The server should then request an updated list
// of roots using the ListRootsRequest.
type RootsListChangedNotification struct {
	Notification
}

func (r *RootsListChangedNotification) GetMethod() RequestMethod {
	return r.Notification.Method
}

func (r *RootsListChangedNotification) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (r *RootsListChangedNotification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in RootsListChangedNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != RootsListChangedNotificationMethod {
		return fmt.Errorf("invalid field method in RootsListChangedNotification: %v", strVal)
	}
	type aliasRootsListChangedNotification RootsListChangedNotification
	var roots aliasRootsListChangedNotification
	if err := json.Unmarshal(b, &roots); err != nil {
		return err
	}
	*r = RootsListChangedNotification(roots)
	return nil
}

type LoggingMessageNotificationParams struct {
	// The severity of this log message.
	Level LoggingLevel `json:"level"`
	// An optional name of the logger issuing this message.
	Logger *string `json:"logger,omitempty"`
	// The data to be logged, such as a string message or an object. Any JSON
	// serializable type is allowed here.
	Data any `json:"data"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (l *LoggingMessageNotificationParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["data"]; raw != nil && !ok {
		return fmt.Errorf("field data in LoggingMessageNotificationParams: required")
	}
	if _, ok := raw["level"]; raw != nil && !ok {
		return fmt.Errorf("field level in LoggingMessageNotificationParams: required")
	}
	type aliasLoggingMessageNotificationParams LoggingMessageNotificationParams
	var params aliasLoggingMessageNotificationParams
	if err := json.Unmarshal(b, &params); err != nil {
		return err
	}
	*l = LoggingMessageNotificationParams(params)
	return nil
}

// Notification of a log message passed from server to client. If no
// logging/setLevel request has been sent from the client, the server MAY decide
// which messages to send automatically.
type LoggingMessageNotification struct {
	Notification
	// Params corresponds to the JSON schema field "params".
	Params LoggingMessageNotificationParams `json:"params"`
}

// LoggingMessageNotification implementation
func (l *LoggingMessageNotification) GetMethod() RequestMethod {
	return l.Notification.Method
}

func (l *LoggingMessageNotification) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (l *LoggingMessageNotification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in LoggingMessageNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != LoggingMessageNotificationMethod {
		return fmt.Errorf("invalid field method in LoggingMessageNotification: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in LoggingMessageNotification: required")
	}
	type aliasLoggingMessageNotification LoggingMessageNotification
	var ln aliasLoggingMessageNotification
	if err := json.Unmarshal(b, &ln); err != nil {
		return err
	}
	*l = LoggingMessageNotification(ln)
	return nil
}

type ResourceUpdatedNotificationParams struct {
	// The URI of the resource that has been updated. This might be a sub-resource of
	// the one that the client actually subscribed to.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *ResourceUpdatedNotificationParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in ResourceUpdatedNotificationParams: required")
	}
	type aliasResourceUpdatedNotificationParams ResourceUpdatedNotificationParams
	var n aliasResourceUpdatedNotificationParams
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*r = ResourceUpdatedNotificationParams(n)
	return nil
}

// A notification from the server to the client, informing it that a resource has
// changed and may need to be read again. This should only be sent if the client
// previously sent a resources/subscribe request.
type ResourceUpdatedNotification struct {
	Notification
	// Params corresponds to the JSON schema field "params".
	Params ResourceUpdatedNotificationParams `json:"params"`
}

// ResourceUpdatedNotification implementation
func (r *ResourceUpdatedNotification) GetMethod() RequestMethod {
	return r.Notification.Method
}

func (r *ResourceUpdatedNotification) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (r *ResourceUpdatedNotification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ResourceUpdatedNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ResourceUpdatedNotificationMethod {
		return fmt.Errorf("invalid field method in ResourceUpdatedNotification: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in ResourceUpdatedNotification: required")
	}
	type aliasResourceUpdatedNotification ResourceUpdatedNotification
	var res aliasResourceUpdatedNotification
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*r = ResourceUpdatedNotification(res)
	return nil
}

// An optional notification from the server to the client, informing it that the
// list of resources it can read from has changed. This may be issued by servers
// without any previous subscription from the client.
type ResourceListChangedNotification struct {
	Notification
}

// ResourceListChangedNotification implementation
func (r *ResourceListChangedNotification) GetMethod() RequestMethod {
	return r.Notification.Method
}

func (r *ResourceListChangedNotification) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (r *ResourceListChangedNotification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ResourceListChangedNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ResourceListChangedNotificationMethod {
		return fmt.Errorf("invalid field method in ResourceListChangedNotification: %v", strVal)
	}
	type aliasResourceListChangedNotification ResourceListChangedNotification
	var res aliasResourceListChangedNotification
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*r = ResourceListChangedNotification(res)
	return nil
}

// An optional notification from the server to the client, informing it that the
// list of tools it offers has changed. This may be issued by servers without any
// previous subscription from the client.
type ToolListChangedNotification struct {
	Notification
}

// ToolListChangedNotification implementation
func (j *ToolListChangedNotification) GetMethod() RequestMethod {
	return j.Notification.Method
}

func (j *ToolListChangedNotification) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ToolListChangedNotification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ToolListChangedNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ToolListChangedNotificationMethod {
		return fmt.Errorf("invalid field method in ToolListChangedNotification: %v", strVal)
	}
	type aliasToolListChangedNotification ToolListChangedNotification
	var tool aliasToolListChangedNotification
	if err := json.Unmarshal(b, &tool); err != nil {
		return err
	}
	*j = ToolListChangedNotification(tool)
	return nil
}

// An optional notification from the server to the client, informing it that the
// list of prompts it offers has changed. This may be issued by servers without any
// previous subscription from the client.
type PromptListChangedNotification struct {
	Notification
}

// PromptListChangedNotification implementation
func (p *PromptListChangedNotification) GetMethod() RequestMethod {
	return p.Notification.Method
}

func (p *PromptListChangedNotification) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (p *PromptListChangedNotification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in PromptListChangedNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != PromptListChangedNotificationMethod {
		return fmt.Errorf("invalid field method in PromptListChangedNotification: %v", strVal)
	}
	type aliasPromptListChangedNotification PromptListChangedNotification
	var n aliasPromptListChangedNotification
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*p = PromptListChangedNotification(n)
	return nil
}

type CancelledNotificationParams[T ID] struct {
	// An optional string describing the reason for the cancellation.
	// This MAY be logged or presented to the user.
	Reason *string `json:"reason,omitempty"`
	// The ID of the request to cancel.
	// This MUST correspond to the ID of a request
	// previously issued in the same direction.
	RequestID RequestID[T] `json:"requestId"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CancelledNotificationParams[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["requestId"]; raw != nil && !ok {
		return fmt.Errorf("field requestId in CancelledNotificationParams: required")
	}
	type aliasCancelledNotificationParams CancelledNotificationParams[T]
	var cn aliasCancelledNotificationParams
	if err := json.Unmarshal(b, &cn); err != nil {
		return err
	}
	*c = CancelledNotificationParams[T](cn)
	return nil
}

// This notification can be sent by either side to indicate that it is cancelling a
// previously-issued request.
//
// The request SHOULD still be in-flight, but due to communication latency, it is
// always possible that this notification MAY arrive after the request has already
// finished.
//
// This notification indicates that the result will be unused, so any associated
// processing SHOULD cease.
//
// A client MUST NOT attempt to cancel its `initialize` request.
type CancelledNotification[T ID] struct {
	Notification
	// Params corresponds to the JSON schema field "params".
	Params CancelledNotificationParams[T] `json:"params"`
}

func (c *CancelledNotification[T]) GetMethod() RequestMethod {
	return c.Notification.Method
}

func (c *CancelledNotification[T]) isNotification() {}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CancelledNotification[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in CancelledNotification: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != CancelledNotificationMethod {
		return fmt.Errorf("invalid field method in CancelledNotification: %v", strVal)
	}
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in CancelledNotification: required")
	}
	type aliasCancelledNotification CancelledNotification[T]
	var cn aliasCancelledNotification
	if err := json.Unmarshal(b, &cn); err != nil {
		return err
	}
	*c = CancelledNotification[T](cn)
	return nil
}

////////////////
/// RESULTS ///
///////////////

// This result property is reserved by the protocol to allow clients and servers to
// attach additional metadata to their responses.
type ResultMeta map[string]any

type Result struct {
	// This result property is reserved by the protocol to allow clients and servers
	// to attach additional metadata to their responses.
	Meta ResultMeta `json:"_meta,omitempty"`
	// AdditionalProperties are reserved for future use.
	AdditionalProperties map[string]any `json:",omitempty"`
}

type PaginatedResult struct {
	// This result property is reserved by the protocol to allow clients and servers
	// to attach additional metadata to their responses.
	Meta ResultMeta `json:"_meta,omitempty"`
	// AdditionalProperties are reserved for future use.
	AdditionalProperties any `json:",omitempty"`
	// An opaque token representing the pagination position after the
	// last returned result. If present, there may be more results available.
	NextCursor *Cursor `json:"nextCursor,omitempty"`
}

// The server's response to a ping request.
type PingResult struct {
	Result
}

func (p *PingResult) isResult() {}

type ContentType string

const (
	ObjectType           ContentType = "object"
	TextContentType      ContentType = "text"
	ImageContentType     ContentType = "image"
	EmbeddedResourceType ContentType = "resource"
)

// NOTE: Go doesn't have sum types so this is a workaround for
// TextResourceContents | BlobResourceContents
type EmbeddedResourceContent struct {
	TextResourceContents *TextResourceContents
	BlobResourceContents *BlobResourceContents
}

// MarshalJSON implements custom JSON marshaling
func (c *EmbeddedResourceContent) MarshalJSON() ([]byte, error) {
	if c.TextResourceContents != nil {
		return json.Marshal(c.TextResourceContents)
	}
	if c.BlobResourceContents != nil {
		return json.Marshal(c.BlobResourceContents)
	}
	return nil, fmt.Errorf("neither TextResourceContents nor BlobResourceContents is set")
}

func (c *EmbeddedResourceContent) UnmarshalJSON(data []byte) error {
	var txt TextResourceContents
	if err := json.Unmarshal(data, &txt); err == nil {
		c.TextResourceContents = &txt
		return nil
	}
	var img BlobResourceContents
	if err := json.Unmarshal(data, &img); err == nil {
		c.BlobResourceContents = &img
		return nil
	}
	return fmt.Errorf("resource matches neither TextResourceContents nor BlobResourceContents")
}

// The contents of a resource, embedded into a prompt or tool call result.
// It is up to the client how best to render embedded resources for the benefit
// of the LLM and/or the user.
type EmbeddedResource struct {
	Annotated
	// Type corresponds to the JSON schema field "type".
	Type ContentType `json:"type"`
	// Resource corresponds to the JSON schema field "resource".
	// TextResourceContents | BlobResourceContents
	Resource EmbeddedResourceContent `json:"resource"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *EmbeddedResource) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["type"]
	if !ok {
		return fmt.Errorf("field type in EmbeddedResource: required")
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid field type in EmbeddedResource: %v", strVal)
	}
	// Validate type is EmbeddedResourceType
	if ContentType(strVal) != EmbeddedResourceType {
		return fmt.Errorf("invalid field type in EmbeddedResource: %v", j.Type)
	}
	type aliasEmbeddedResource EmbeddedResource
	var res aliasEmbeddedResource
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*j = EmbeddedResource(res)
	return nil
}

// Describes the name and version of an MCP implementation.
type Implementation struct {
	// Name corresponds to the JSON schema field "name".
	Name string `json:"name"`
	// Version corresponds to the JSON schema field "version".
	Version string `json:"version"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Implementation) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in Implementation: required")
	}
	if _, ok := raw["version"]; raw != nil && !ok {
		return fmt.Errorf("field version in Implementation: required")
	}
	type aliasImplementation Implementation
	var impl aliasImplementation
	if err := json.Unmarshal(b, &impl); err != nil {
		return err
	}
	*j = Implementation(impl)
	return nil
}

// NOTE: Go doesn't have sum types so this is a workaround for
// TextContent | ImageContent | EmbeddedResource
type PromptMessageContent struct {
	TextContent      *TextContent
	ImageContent     *ImageContent
	EmbeddedResource *EmbeddedResource
}

// MarshalJSON implements custom JSON marshaling
func (p *PromptMessageContent) MarshalJSON() ([]byte, error) {
	if p.TextContent != nil {
		if p.TextContent.Type != TextContentType {
			return nil, fmt.Errorf("invalid content type for TextConten: %q", p.TextContent.Type)
		}
		return json.Marshal(p.TextContent)
	}
	if p.ImageContent != nil {
		if p.ImageContent.Type != ImageContentType {
			return nil, fmt.Errorf("invalid content type for ImageContent: %q", p.ImageContent.Type)
		}
		return json.Marshal(p.ImageContent)
	}
	if p.EmbeddedResource != nil {
		if p.EmbeddedResource.Type != EmbeddedResourceType {
			return nil, fmt.Errorf("invalid content type for EmbeddedResource: %q", p.EmbeddedResource.Type)
		}
		return json.Marshal(p.EmbeddedResource)
	}
	return nil, fmt.Errorf("neither TextContent, ImageContent nor EmbeddedResource is set")
}

func (p *PromptMessageContent) UnmarshalJSON(data []byte) error {
	var txt TextContent
	if err := json.Unmarshal(data, &txt); err == nil && txt.Type == TextContentType {
		p.TextContent = &txt
		return nil
	}
	var img ImageContent
	if err := json.Unmarshal(data, &img); err == nil && img.Type == ImageContentType {
		p.ImageContent = &img
		return nil
	}
	var res EmbeddedResource
	if err := json.Unmarshal(data, &res); err == nil && res.Type == EmbeddedResourceType {
		p.EmbeddedResource = &res
		return nil
	}
	return fmt.Errorf("result content matches neither of TextContent, ImageContent, EmbeddedResource")
}

// Describes a message returned as part of a prompt.
// This is similar to `SamplingMessage`, but also supports the embedding of
// resources from the MCP server.
type PromptMessage struct {
	// Role corresponds to the JSON schema field "role".
	Role Role `json:"role"`
	// Content corresponds to the JSON schema field "content".
	// TextContent | ImageContent | EmbeddedResource
	Content PromptMessageContent `json:"content"`
}

func (j *PromptMessage) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["content"]; raw != nil && !ok {
		return fmt.Errorf("field content in SamplingMessage: required")
	}
	val, ok := raw["role"]
	if !ok {
		return fmt.Errorf("field role in SamplingMessage: required")
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid field role in SamplingMessage: %v", strVal)
	}
	if _, valid := enumRole[Role(strVal)]; !valid {
		return fmt.Errorf("invalid SamplingMessage role value: %v", strVal)
	}
	type aliasPromptMessage PromptMessage
	var p aliasPromptMessage
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*j = PromptMessage(p)
	return nil
}

// Identifies a prompt.
type PromptReference struct {
	// Type corresponds to the JSON schema field "type".
	Type ReferenceType `json:"type"`
	// The name of the prompt or prompt template
	Name string `json:"name"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PromptReference) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in PromptReference: required")
	}
	val, ok := raw["type"]
	if raw != nil && !ok {
		return fmt.Errorf("field type in PromptReference: required")
	}
	if strVal, ok := val.(string); !ok || ReferenceType(strVal) != PromptReferenceType {
		return fmt.Errorf("invalid field type in PromptReference: %v", strVal)
	}
	type aliasPromptReference PromptReference
	var ref aliasPromptReference
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	*j = PromptReference(ref)
	return nil
}

// A known resource that the server is capable of reading.
type Resource struct {
	Annotated
	// A description of what this resource represents.
	// This can be used by clients to improve the LLM's understanding of available
	// resources. It can be thought of like a "hint" to the model.
	Description *string `json:"description,omitempty"`
	// The MIME type of this resource, if known.
	MimeType *string `json:"mimeType,omitempty"`
	// A human-readable name for this resource.
	// This can be used by clients to populate UI elements.
	Name string `json:"name"`
	// The URI of this resource.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Resource) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in Resource: required")
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in Resource: required")
	}
	type aliasResource Resource
	var res aliasResource
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*r = Resource(res)
	return nil
}

// The contents of a specific resource or sub-resource.
type ResourceContents struct {
	// The MIME type of this resource, if known.
	MimeType *string `json:"mimeType,omitempty"`
	// The URI of this resource.
	URI string `json:"uri"`
}

// A reference to a resource or resource template definition.
type ResourceReference struct {
	// Type corresponds to the JSON schema field "type".
	Type ReferenceType `json:"type"`
	// The URI or URI template of the resource.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *ResourceReference) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["type"]
	if raw != nil && !ok {
		return fmt.Errorf("field type in ResourceReference: required")
	}
	if strVal, ok := val.(string); !ok || ReferenceType(strVal) != ResourceReferenceType {
		return fmt.Errorf("invalid field type in ResourceReference: %v", strVal)
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in ResourceReference: required")
	}
	type aliasResourceReference ResourceReference
	var res aliasResourceReference
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*r = ResourceReference(res)
	return nil
}

type Role string

const (
	RoleAssistant Role = "assistant"
	RoleUser      Role = "user"
)

var enumRole = map[Role]struct{}{
	RoleAssistant: {},
	RoleUser:      {},
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Role) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	role := Role(s)
	if _, valid := enumRole[role]; !valid {
		return fmt.Errorf("invalid role: %q", s)
	}

	*r = role
	return nil
}

// The client's response to a sampling/create_message request from the server. The
// client should inform the user before returning the sampled message, to allow
// them to inspect the response (human in the loop) and decide whether to allow the
// server to see it.
type CreateMessageResult struct {
	Result
	// Sampling message content
	Content SamplingMessageContent `json:"content"`
	// The name of the model that generated the message.
	Model string `json:"model"`
	// Role corresponds to the JSON schema field "role".
	Role Role `json:"role"`
	// The reason why sampling stopped, if known.
	StopReason *string `json:"stopReason,omitempty"`
}

func (c *CreateMessageResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CreateMessageResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["content"]; raw != nil && !ok {
		return fmt.Errorf("field content in CreateMessageResult: required")
	}
	if _, ok := raw["model"]; raw != nil && !ok {
		return fmt.Errorf("field model in CreateMessageResult: required")
	}
	val, ok := raw["role"]
	if !ok {
		return fmt.Errorf("field role in SamplingMessage: required")
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid field role in SamplingMessage: %v", strVal)
	}
	if _, valid := enumRole[Role(strVal)]; !valid {
		return fmt.Errorf("invalid SamplingMessage role value: %v", strVal)
	}
	type aliasCreateMessageResult CreateMessageResult
	var cm aliasCreateMessageResult
	if err := json.Unmarshal(b, &cm); err != nil {
		return err
	}
	*c = CreateMessageResult(cm)
	return nil
}

// Represents a root directory or file that the server can operate on.
type Root struct {
	// An optional name for the root. This can be used to provide a human-readable
	// identifier for the root, which may be useful for display purposes or for
	// referencing the root in other parts of the application.
	Name *string `json:"name,omitempty"`
	// The URI identifying the root. This *must* start with file:// for now.
	// This restriction may be relaxed in future versions of the protocol to allow
	// other URI schemes.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Root) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in Root: required")
	}
	type aliasRoot Root
	var root aliasRoot
	if err := json.Unmarshal(b, &root); err != nil {
		return err
	}
	*r = Root(root)
	return nil
}

// The client's response to a roots/list request from the server.
// This result contains an array of Root objects, each representing
// a root directory or file that the server can operate on.
type ListRootsResult struct {
	Result
	// Roots corresponds to the JSON schema field "roots".
	Roots []Root `json:"roots"`
}

func (l *ListRootsResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ListRootsResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["roots"]; raw != nil && !ok {
		return fmt.Errorf("field roots in ListRootsResult: required")
	}
	type aliasListRootsResult ListRootsResult
	var res aliasListRootsResult
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*l = ListRootsResult(res)
	return nil
}

type CompleteResultCompletion struct {
	// Indicates whether there are additional completion options beyond those provided
	// in the current response, even if the exact total is unknown.
	HasMore *bool `json:"hasMore,omitempty"`
	// The total number of completion options available. This can exceed the number of
	// values actually sent in the response.
	Total *int `json:"total,omitempty"`
	// An array of completion values. Must not exceed 100 items.
	Values []string `json:"values"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CompleteResultCompletion) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["values"]; raw != nil && !ok {
		return fmt.Errorf("field values in CompleteResultCompletion: required")
	}
	type aliasCompleteResultCompletion CompleteResultCompletion
	var cr aliasCompleteResultCompletion
	if err := json.Unmarshal(b, &cr); err != nil {
		return err
	}
	*c = CompleteResultCompletion(cr)
	return nil
}

// The server's response to a completion/complete request
type CompleteResult struct {
	Result
	// Completion corresponds to the JSON schema field "completion".
	Completion CompleteResultCompletion `json:"completion"`
}

func (j *CompleteResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CompleteResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["completion"]; raw != nil && !ok {
		return fmt.Errorf("field completion in CompleteResult: required")
	}
	type aliasCompleteResult CompleteResult
	var c aliasCompleteResult
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}
	*j = CompleteResult(c)
	return nil
}

// After receiving an initialize request from the client, the server sends this
// response.
type InitializeResult struct {
	Result
	// Capabilities corresponds to the JSON schema field "capabilities".
	Capabilities ServerCapabilities `json:"capabilities"`
	// Instructions describing how to use the server and its features.
	// This can be used by clients to improve the LLM's understanding of available
	// tools, resources, etc. It can be thought of like a "hint" to the model. For
	// example, this information MAY be added to the system prompt.
	Instructions *string `json:"instructions,omitempty"`
	// The version of the Model Context Protocol that the server wants to use. This
	// may not match the version that the client requested. If the client cannot
	// support this version, it MUST disconnect.
	ProtocolVersion string `json:"protocolVersion"`
	// ServerInfo corresponds to the JSON schema field "serverInfo".
	ServerInfo Implementation `json:"serverInfo"`
}

func (j *InitializeResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InitializeResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["capabilities"]; raw != nil && !ok {
		return fmt.Errorf("field capabilities in InitializeResult: required")
	}
	if _, ok := raw["protocolVersion"]; raw != nil && !ok {
		return fmt.Errorf("field protocolVersion in InitializeResult: required")
	}
	if _, ok := raw["serverInfo"]; raw != nil && !ok {
		return fmt.Errorf("field serverInfo in InitializeResult: required")
	}
	type aliasInitializeResult InitializeResult
	var res aliasInitializeResult
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*j = InitializeResult(res)
	return nil
}

// The server's response to a prompts/get request from the client.
type GetPromptResult struct {
	Result
	// An optional description for the prompt.
	Description *string `json:"description,omitempty"`
	// Messages corresponds to the JSON schema field "messages".
	Messages []PromptMessage `json:"messages"`
}

func (g *GetPromptResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (g *GetPromptResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["messages"]; raw != nil && !ok {
		return fmt.Errorf("field messages in GetPromptResult: required")
	}
	type aliasGetPromptResult GetPromptResult
	var res aliasGetPromptResult
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*g = GetPromptResult(res)
	return nil
}

// Describes an argument that a prompt can accept.
type PromptArgument struct {
	// The name of the argument.
	Name string `json:"name"`
	// A human-readable description of the argument.
	Description *string `json:"description,omitempty"`
	// Whether this argument must be provided.
	Required *bool `json:"required,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *PromptArgument) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in PromptArgument: required")
	}
	type aliasPromptArgument PromptArgument
	var arg aliasPromptArgument
	if err := json.Unmarshal(b, &arg); err != nil {
		return err
	}
	*p = PromptArgument(arg)
	return nil
}

// A prompt or prompt template that the server offers.
type Prompt struct {
	// The name of the prompt or prompt template.
	Name string `json:"name"`
	// An optional description of what this prompt provides
	Description *string `json:"description,omitempty"`
	// A list of arguments to use for templating the prompt.
	Arguments []PromptArgument `json:"arguments,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *Prompt) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in Prompt: required")
	}
	type aliasPrompt Prompt
	var prompt aliasPrompt
	if err := json.Unmarshal(b, &prompt); err != nil {
		return err
	}
	*p = Prompt(prompt)
	return nil
}

// The server's response to a prompts/list request from the client.
type ListPromptsResult struct {
	PaginatedResult
	// Prompts corresponds to the JSON schema field "prompts".
	Prompts []Prompt `json:"prompts"`
}

func (l *ListPromptsResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ListPromptsResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["prompts"]; raw != nil && !ok {
		return fmt.Errorf("field prompts in ListPromptsResult: required")
	}
	type aliasListPromptsResult ListPromptsResult
	var res aliasListPromptsResult
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*l = ListPromptsResult(res)
	return nil
}

// The server's response to a resources/list request from the client.
type ListResourcesResult struct {
	PaginatedResult
	// Resources corresponds to the JSON schema field "resources".
	Resources []Resource `json:"resources"`
}

func (l *ListResourcesResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ListResourcesResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["resources"]; raw != nil && !ok {
		return fmt.Errorf("field resources in ListResourcesResult: required")
	}
	type aliasListResourcesResult ListResourcesResult
	var res aliasListResourcesResult
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*l = ListResourcesResult(res)
	return nil
}

// A template description for resources available on the server.
type ResourceTemplate struct {
	Annotated
	// A description of what this template is for.
	// This can be used by clients to improve the LLM's understanding of available
	// resources. It can be thought of like a "hint" to the model.
	Description *string `json:"description,omitempty"`
	// The MIME type for all resources that match this template. This should only be
	// included if all resources matching this template have the same type.
	MimeType *string `json:"mimeType,omitempty"`
	// A human-readable name for the type of resource this template refers to.
	// This can be used by clients to populate UI elements.
	Name string `json:"name"`
	// A URI template (according to RFC 6570) that can be used to construct resource
	URITemplate string `json:"uriTemplate"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (l *ResourceTemplate) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in ResourceTemplate: required")
	}
	if _, ok := raw["uriTemplate"]; raw != nil && !ok {
		return fmt.Errorf("field uriTemplate in ResourceTemplate: required")
	}
	type aliasResourceTemplate ResourceTemplate
	var res aliasResourceTemplate
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*l = ResourceTemplate(res)
	return nil
}

// The server's response to a resources/templates/list request from the client.
type ListResourceTemplatesResult struct {
	PaginatedResult
	// ResourceTemplates corresponds to the JSON schema field "resourceTemplates".
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

func (j *ListResourceTemplatesResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListResourceTemplatesResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["resourceTemplates"]; raw != nil && !ok {
		return fmt.Errorf("field resourceTemplates in ListResourceTemplatesResult: required")
	}
	type aliasListResourceTemplatesResult ListResourceTemplatesResult
	var res aliasListResourceTemplatesResult
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*j = ListResourceTemplatesResult(res)
	return nil
}

// NOTE: Go doesn't have sum types so this is a workaround for
// TextResourceContents | BlobResourceContents
type ReadResourceResultContent struct {
	TextResourceContents *TextResourceContents
	BlobResourceContents *BlobResourceContents
}

func (c *ReadResourceResultContent) MarshalJSON() ([]byte, error) {
	if c.TextResourceContents != nil {
		return json.Marshal(c.TextResourceContents)
	}
	if c.BlobResourceContents != nil {
		return json.Marshal(c.BlobResourceContents)
	}
	return nil, fmt.Errorf("neither TextResourceContents nor BlobResourceContents is set")
}

func (c *ReadResourceResultContent) UnmarshalJSON(data []byte) error {
	var txt TextResourceContents
	if err := json.Unmarshal(data, &txt); err == nil {
		c.TextResourceContents = &txt
		return nil
	}
	var blob BlobResourceContents
	if err := json.Unmarshal(data, &blob); err == nil {
		c.BlobResourceContents = &blob
		return nil
	}
	return fmt.Errorf("content matches neither TextResourceContents nor BlobResourceContents")
}

// The server's response to a resources/read request from the client.
type ReadResourceResult struct {
	Result
	// Contents corresponds to the JSON schema field "contents".
	// TextResourceContents | BlobResourceContents
	Contents []ReadResourceResultContent `json:"contents"`
}

func (j *ReadResourceResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ReadResourceResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["contents"]; raw != nil && !ok {
		return fmt.Errorf("field content in SamplingMessage: required")
	}
	type aliasReadResourceResult ReadResourceResult
	var r aliasReadResourceResult
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}
	*j = ReadResourceResult(r)
	return nil
}

type AnnotatedAnnotations struct {
	// Describes who the intended customer of this object or data is.
	// It can include multiple entries to indicate content useful for multiple
	// audiences (e.g., `["user", "assistant"]`).
	Audience []Role `json:"audience,omitempty"`
	// Describes how important this data is for operating the server.
	// A value of 1 means "most important," and indicates that the data is
	// effectively required, while 0 means "least important," and indicates that
	// the data is entirely optional.
	Priority *int64 `json:"priority,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AnnotatedAnnotations) UnmarshalJSON(b []byte) error {
	var a AnnotatedAnnotations
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	if a.Priority != nil && 1 < *a.Priority {
		return fmt.Errorf("field %s: must be <= %v", "priority", 1)
	}
	if a.Priority != nil && 0 > *a.Priority {
		return fmt.Errorf("field %s: must be >= %v", "priority", 0)
	}
	*j = a
	return nil
}

// Base for objects that include optional annotations for the client. The client
// can use annotations to inform how objects are used or displayed
type Annotated struct {
	// Annotations corresponds to the JSON schema field "annotations".
	Annotations *AnnotatedAnnotations `json:"annotations,omitempty"`
}

// NOTE: Go doesn't have sum types so this is a workaround for
// TextContent | ImageContent | EmbeddedResource
type CallToolResultContent struct {
	TextContent      *TextContent
	ImageContent     *ImageContent
	EmbeddedResource *EmbeddedResource
}

// MarshalJSON implements custom JSON marshaling
func (c *CallToolResultContent) MarshalJSON() ([]byte, error) {
	if c.TextContent != nil {
		if c.TextContent.Type != TextContentType {
			return nil, fmt.Errorf("invalid content type for TextConten: %q", c.TextContent.Type)
		}
		return json.Marshal(c.TextContent)
	}
	if c.ImageContent != nil {
		if c.ImageContent.Type != ImageContentType {
			return nil, fmt.Errorf("invalid content type for ImageContent: %q", c.ImageContent.Type)
		}
		return json.Marshal(c.ImageContent)
	}
	if c.EmbeddedResource != nil {
		if c.EmbeddedResource.Type != EmbeddedResourceType {
			return nil, fmt.Errorf("invalid content type for EmbeddedResource: %q", c.EmbeddedResource.Type)
		}
		return json.Marshal(c.EmbeddedResource)
	}
	return nil, fmt.Errorf("neither TextContent, ImageContent nor EmbeddedResource is set")
}

func (c *CallToolResultContent) UnmarshalJSON(data []byte) error {
	type aliasTextContent TextContent
	var txt aliasTextContent
	if err := json.Unmarshal(data, &txt); err == nil && txt.Type == TextContentType {
		textContent := TextContent(txt)
		c.TextContent = &textContent
		return nil
	}

	type aliasImgContent ImageContent
	var img aliasImgContent
	if err := json.Unmarshal(data, &img); err == nil && img.Type == ImageContentType {
		imgContent := ImageContent(img)
		c.ImageContent = &imgContent
		return nil
	}

	type aliasEmbContent EmbeddedResource
	var res aliasEmbContent
	if err := json.Unmarshal(data, &res); err == nil && res.Type == EmbeddedResourceType {
		embRes := EmbeddedResource(res)
		c.EmbeddedResource = &embRes
		return nil
	}

	return fmt.Errorf("result content matches neither of TextContent, ImageContent, EmbeddedResource")
}

// The server's response to a tool call.
//
// Any errors that originate from the tool SHOULD be reported inside the result
// object, with `isError` set to true, _not_ as an MCP protocol-level error
// response. Otherwise, the LLM would not be able to see that an error occurred
// and self-correct.
//
// However, any errors in _finding_ the tool, an error indicating that the
// server does not support tool calls, or any other exceptional conditions,
// should be reported as an MCP error response.
type CallToolResult struct {
	Result
	// Content corresponds to the JSON schema field "content".
	// TextContent | ImageContent | EmbeddedResource
	Content []CallToolResultContent `json:"content"`
	// Whether the tool call ended in an error.
	// If not set, this is assumed to be false (the call was successful).
	IsError *bool `json:"isError,omitempty"`
}

func (j *CallToolResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CallToolResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["content"]; raw != nil && !ok {
		return fmt.Errorf("field content in SamplingMessage: required")
	}
	type aliasCallToolResult CallToolResult
	var c aliasCallToolResult
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}
	*j = CallToolResult(c)
	return nil
}

type ToolInputSchemaProperties map[string]map[string]any

// A JSON Schema object defining the expected parameters for the tool.
type ToolInputSchema struct {
	// Type corresponds to the JSON schema field "type".
	Type ContentType `json:"type"`
	// Properties corresponds to the JSON schema field "properties".
	Properties ToolInputSchemaProperties `json:"properties,omitempty"`
}

// Definition for a tool the client can call.
type Tool struct {
	// The name of the tool.
	Name string `json:"name"`
	// A human-readable description of the tool.
	Description *string `json:"description,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema ToolInputSchema `json:"inputSchema"`
}

// The server's response to a tools/list request from the client.
type ListToolsResult struct {
	PaginatedResult
	// Tools corresponds to the JSON schema field "tools".
	Tools []Tool `json:"tools"`
}

func (j *ListToolsResult) isResult() {}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListToolsResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["tools"]; raw != nil && !ok {
		return fmt.Errorf("field tools in ListToolsResult: required")
	}
	type aliasListToolsResult ListToolsResult
	var res aliasListToolsResult
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	*j = ListToolsResult(res)
	return nil
}

// Experimental, non-standard capabilities that the server supports.
type ServerCapabilitiesExperimental map[string]map[string]any

// Present if the server supports sending log messages to the client.
type ServerCapabilitiesLogging map[string]any

// Present if the server offers any prompt templates.
type ServerCapabilitiesPrompts struct {
	// Whether this server supports notifications for changes to the prompt list.
	ListChanged *bool `json:"listChanged,omitempty"`
}

// Present if the server offers any resources to read.
type ServerCapabilitiesResources struct {
	// Whether this server supports notifications for changes to the resource list.
	ListChanged *bool `json:"listChanged,omitempty"`
	// Whether this server supports subscribing to resource updates.
	Subscribe *bool `json:"subscribe,omitempty"`
}

// Present if the server offers any tools to call.
type ServerCapabilitiesTools struct {
	// Whether this server supports notifications for changes to the tool list.
	ListChanged *bool `json:"listChanged,omitempty"`
}

// Capabilities that a server may support. Known capabilities are defined here, in
// this schema, but this is not a closed set: any server can define its own,
// additional capabilities.
type ServerCapabilities struct {
	// Experimental, non-standard capabilities that the server supports.
	Experimental ServerCapabilitiesExperimental `json:"experimental,omitempty"`
	// Present if the server supports sending log messages to the client.
	Logging ServerCapabilitiesLogging `json:"logging,omitempty"`
	// Present if the server offers any prompt templates.
	Prompts *ServerCapabilitiesPrompts `json:"prompts,omitempty"`
	// Present if the server offers any resources to read.
	Resources *ServerCapabilitiesResources `json:"resources,omitempty"`
	// Present if the server offers any tools to call.
	Tools *ServerCapabilitiesTools `json:"tools,omitempty"`
}

// An image provided to or from an LLM.
type ImageContent struct {
	Annotated
	// The base64-encoded image data.
	Data string `json:"data"`
	// The MIME type of the image. Different providers may support different image types.
	MimeType string `json:"mimeType"`
	// Type corresponds to the JSON schema field "type".
	Type ContentType `json:"type"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ImageContent) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["data"]; raw != nil && !ok {
		return fmt.Errorf("field data in ImageContent: required")
	}
	if _, ok := raw["mimeType"]; raw != nil && !ok {
		return fmt.Errorf("field mimeType in ImageContent: required")
	}
	val, ok := raw["type"]
	if raw != nil && !ok {
		return fmt.Errorf("field type in ImageContent: required")
	}
	if strVal, ok := val.(string); !ok || ContentType(strVal) != ImageContentType {
		return fmt.Errorf("invalid field type in ImageContent: %v", strVal)
	}
	type aliasImageContent ImageContent
	var img aliasImageContent
	if err := json.Unmarshal(b, &img); err != nil {
		return err
	}
	*j = ImageContent(img)
	return nil
}

// Text provided to or from an LLM.
type TextContent struct {
	Annotated
	// The text content of the message.
	Text string `json:"text"`
	// Type corresponds to the JSON schema field "type".
	Type ContentType `json:"type"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *TextContent) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["text"]; raw != nil && !ok {
		return fmt.Errorf("field text in TextContent: required")
	}
	val, ok := raw["type"]
	if raw != nil && !ok {
		return fmt.Errorf("field type in TextContent: required")
	}
	if strVal, ok := val.(string); !ok || ContentType(strVal) != TextContentType {
		return fmt.Errorf("invalid field type in TextContent: %v", strVal)
	}
	type aliasTextContent TextContent
	var text aliasTextContent
	if err := json.Unmarshal(b, &text); err != nil {
		return err
	}
	*j = TextContent(text)
	return nil
}

type TextResourceContents struct {
	ResourceContents
	// The text of the item. This must only be set if the item can actually be
	// represented as text (not binary data).
	Text string `json:"text"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *TextResourceContents) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["text"]; raw != nil && !ok {
		return fmt.Errorf("field text in TextResourceContents: required")
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in TextResourceContents: required")
	}
	type aliasTextResourceContents TextResourceContents
	var text aliasTextResourceContents
	if err := json.Unmarshal(b, &text); err != nil {
		return err
	}
	*t = TextResourceContents(text)
	return nil
}

type BlobResourceContents struct {
	ResourceContents
	// A base64-encoded string representing the binary data of the item.
	Blob string `json:"blob"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *BlobResourceContents) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["blob"]; raw != nil && !ok {
		return fmt.Errorf("field blob in BlobResourceContents: required")
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in BlobResourceContents: required")
	}
	type aliasBlobResourceContents BlobResourceContents
	var blob aliasBlobResourceContents
	if err := json.Unmarshal(b, &blob); err != nil {
		return err
	}
	*j = BlobResourceContents(blob)
	return nil
}
