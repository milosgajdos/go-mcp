package mcp

import (
	"encoding/json"
	"fmt"
)

// Simple constraint for string|number unions
type StringOrNumber interface {
	~string | ~int
}

// Token is used as a generic constraint.
// NOTE: this is mostly to make the semantics clearer
// to the readers; the interface could be extended, too.
type Token StringOrNumber

// A progress token, used to associate progress
// notifications with the original request.
// NOTE we could also define a struct like this:
//
//	type UnionValue[T StringOrNumber] struct {
//	    Value T
//	}
//
// # And then then ProgressToken like so
//
// type ProgressToken = UnionValue[StringOrNumber]
type ProgressToken[T StringOrNumber] struct {
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
		return fmt.Errorf("failed to unmarshal union value: %w", err)
	}
	p.Value = v
	return nil
}

// An opaque token used to represent a cursor for pagination.
type Cursor string

// Base for objects that include optional annotations for the client. The client
// can use annotations to inform how objects are used or displayed
type Annotated struct {
	// Annotations corresponds to the JSON schema field "annotations".
	Annotations *AnnotatedAnnotations `json:"annotations,omitempty"`
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
	type Plain AnnotatedAnnotations
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.Priority != nil && 1 < *plain.Priority {
		return fmt.Errorf("field %s: must be <= %v", "priority", 1)
	}
	if plain.Priority != nil && 0 > *plain.Priority {
		return fmt.Errorf("field %s: must be >= %v", "priority", 0)
	}
	*j = AnnotatedAnnotations(plain)
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
func (j *CallToolRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in CallToolRequestParams: required")
	}
	type Plain CallToolRequestParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CallToolRequestParams(plain)
	return nil
}

// Used by the client to invoke a tool provided by the server.
type CallToolRequest[T Token] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params CallToolRequestParams `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CallToolRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain CallToolRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CallToolRequest[T](plain)
	return nil
}

type CallToolResultContent interface {
	CallToolResultContentType() ContentType
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

// UnmarshalJSON implements json.Unmarshaler.
func (j *CallToolResult) UnmarshalJSON(b []byte) error {
	type Alias CallToolResult
	aux := &struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	if aux.Content == nil {
		return fmt.Errorf("field content in CallToolResult: required")
	}

	j.Content = make([]CallToolResultContent, len(aux.Content))
	for i, raw := range aux.Content {
		var text TextContent
		if err := json.Unmarshal(raw, &text); err == nil {
			j.Content[i] = &text
			continue
		}

		var image ImageContent
		if err := json.Unmarshal(raw, &image); err == nil {
			j.Content[i] = &image
			continue
		}

		var embedded EmbeddedResource
		if err := json.Unmarshal(raw, &embedded); err == nil {
			j.Content[i] = &embedded
			continue
		}

		return fmt.Errorf("content at index %d matches neither TextContent, ImageContent, nor EmbeddedResource", i)
	}

	return nil
}

type CancelledNotificationParams[T ID] struct {
	// An optional string describing the reason for the cancellation. This MAY be
	// logged or presented to the user.
	Reason *string `json:"reason,omitempty"`
	// The ID of the request to cancel.
	//
	// This MUST correspond to the ID of a request previously issued in the same
	// direction.
	RequestID RequestID[T] `json:"requestId"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CancelledNotificationParams[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["requestId"]; raw != nil && !ok {
		return fmt.Errorf("field requestId in CancelledNotificationParams: required")
	}
	type Plain CancelledNotificationParams[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CancelledNotificationParams[T](plain)
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
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params CancelledNotificationParams[T] `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CancelledNotification[T]) UnmarshalJSON(b []byte) error {
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
	type Plain CancelledNotification[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CancelledNotification[T](plain)
	return nil
}

type ClientCapabilitiesExperimental map[string]map[string]any

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
	type Plain CompleteRequestParamsArgument
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CompleteRequestParamsArgument(plain)
	return nil
}

type ReferenceType string

const (
	PromptReferenceType   ReferenceType = "ref/prompt"
	ResourceReferenceType ReferenceType = "ref/resource"
)

type ContentType string

const (
	ObjectType           ContentType = "object"
	TextContentType      ContentType = "text"
	ImageContentType     ContentType = "image"
	EmbeddedResourceType ContentType = "resource"
)

type ResourceContentType string

const (
	TextResourceContentsType ResourceContentType = "text"
	BlobResourceContentsType ResourceContentType = "blob"
)

type CompleteRequestParamsRef interface {
	CompleteRequestParamsRefType() ReferenceType
}

type CompleteRequestParams struct {
	// Ref corresponds to the JSON schema field "ref".
	// PromptReference | ResourceReference
	Ref CompleteRequestParamsRef `json:"ref"`
	// The argument's information
	Argument CompleteRequestParamsArgument `json:"argument"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CompleteRequestParams) UnmarshalJSON(b []byte) error {
	type Alias CompleteRequestParams
	aux := &struct {
		Ref json.RawMessage `json:"ref"`
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	if aux.Ref == nil {
		return fmt.Errorf("field ref in CompleteRequestParams: required")
	}

	// Try each possible ref type
	var promptRef PromptReference
	if err := json.Unmarshal(aux.Ref, &promptRef); err == nil {
		j.Ref = &promptRef
		return nil
	}

	var resourceRef ResourceReference
	if err := json.Unmarshal(aux.Ref, &resourceRef); err == nil {
		j.Ref = &resourceRef
		return nil
	}

	return fmt.Errorf("ref matches neither PromptReference nor ResourceReference")
}

// A request from the client to the server, to ask for completion options.
type CompleteRequest[T Token] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params CompleteRequestParams `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CompleteRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain CompleteRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CompleteRequest[T](plain)
	return nil
}

// The server's response to a completion/complete request
type CompleteResult struct {
	Result
	// Completion corresponds to the JSON schema field "completion".
	Completion CompleteResultCompletion `json:"completion"`
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
func (j *CompleteResultCompletion) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["values"]; raw != nil && !ok {
		return fmt.Errorf("field values in CompleteResultCompletion: required")
	}
	type Plain CompleteResultCompletion
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CompleteResultCompletion(plain)
	return nil
}

// This result property is reserved by the protocol to allow clients and servers to
// attach additional metadata to their responses.
type CompleteResultMeta map[string]any

// UnmarshalJSON implements json.Unmarshaler.
func (j *CompleteResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["completion"]; raw != nil && !ok {
		return fmt.Errorf("field completion in CompleteResult: required")
	}
	type Plain CompleteResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CompleteResult(plain)
	return nil
}

type CreateMessageRequestParamsIncludeContext string

const (
	CreateMessageRequestParamsIncludeContextNone       CreateMessageRequestParamsIncludeContext = "none"
	CreateMessageRequestParamsIncludeContextAllServers CreateMessageRequestParamsIncludeContext = "allServers"
	CreateMessageRequestParamsIncludeContextThisServer CreateMessageRequestParamsIncludeContext = "thisServer"
)

var enumValuesCreateMessageRequestParamsIncludeContext = map[CreateMessageRequestParamsIncludeContext]struct{}{
	CreateMessageRequestParamsIncludeContextNone:       {},
	CreateMessageRequestParamsIncludeContextAllServers: {},
	CreateMessageRequestParamsIncludeContextThisServer: {},
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CreateMessageRequestParamsIncludeContext) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	ctx := CreateMessageRequestParamsIncludeContext(v)
	if _, valid := enumValuesCreateMessageRequestParamsIncludeContext[ctx]; !valid {
		return fmt.Errorf("invalid CreateMessageRequestParamsIncludeContext value: %v", ctx)
	}
	*j = ctx
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
	IncludeContext *CreateMessageRequestParamsIncludeContext `json:"includeContext,omitempty"`
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
func (j *CreateMessageRequestParams) UnmarshalJSON(b []byte) error {
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
	type Plain CreateMessageRequestParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CreateMessageRequestParams(plain)
	return nil
}

// A request from the server to sample an LLM via the client. The client has full
// discretion over which model to select. The client should also inform the user
// before beginning sampling, to allow them to inspect the request (human in the
// loop) and decide whether to approve it.
type CreateMessageRequest[T Token] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params CreateMessageRequestParams `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CreateMessageRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain CreateMessageRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CreateMessageRequest[T](plain)
	return nil
}

// The client's response to a sampling/create_message request from the server. The
// client should inform the user before returning the sampled message, to allow
// them to inspect the response (human in the loop) and decide whether to allow the
// server to see it.
type CreateMessageResult struct {
	Result
	SamplingMessage
	// The name of the model that generated the message.
	Model string `json:"model"`
	// The reason why sampling stopped, if known.
	StopReason *string `json:"stopReason,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *CreateMessageResult) UnmarshalJSON(b []byte) error {
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
	if _, ok := raw["role"]; raw != nil && !ok {
		return fmt.Errorf("field role in CreateMessageResult: required")
	}
	type Plain CreateMessageResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = CreateMessageResult(plain)
	return nil
}

type EmbeddedResourceContent interface {
	EmbeddedResourceContentType() ResourceContentType
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
	type Alias EmbeddedResource
	aux := &struct {
		Resource json.RawMessage `json:"resource"`
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	if aux.Resource == nil {
		return fmt.Errorf("field resource in EmbeddedResource: required")
	}

	// Validate type is EmbeddedResourceType
	if j.Type != EmbeddedResourceType {
		return fmt.Errorf("invalid field type in EmbeddedResource: %v", j.Type)
	}

	// Try each possible resource type
	var text TextResourceContents
	if err := json.Unmarshal(aux.Resource, &text); err == nil {
		j.Resource = &text
		return nil
	}

	var blob BlobResourceContents
	if err := json.Unmarshal(aux.Resource, &blob); err == nil {
		j.Resource = &blob
		return nil
	}

	return fmt.Errorf("resource matches neither TextResourceContents nor BlobResourceContents")
}

func (j EmbeddedResource) CallToolResultContentType() ContentType {
	return EmbeddedResourceType
}

func (j EmbeddedResource) PromptMessageContentType() ContentType {
	return EmbeddedResourceType
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
func (j *GetPromptRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in GetPromptRequestParams: required")
	}
	type Plain GetPromptRequestParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = GetPromptRequestParams(plain)
	return nil
}

// Used by the client to get a prompt provided by the server.
type GetPromptRequest[T Token] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params GetPromptRequestParams `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *GetPromptRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain GetPromptRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = GetPromptRequest[T](plain)
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

// UnmarshalJSON implements json.Unmarshaler.
func (j *GetPromptResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["messages"]; raw != nil && !ok {
		return fmt.Errorf("field messages in GetPromptResult: required")
	}
	type Plain GetPromptResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = GetPromptResult(plain)
	return nil
}

// An image provided to or from an LLM.
type ImageContent struct {
	Annotated
	// The base64-encoded image data.
	Data string `json:"data"`
	// The MIME type of the image. Different providers may support different image
	// types.
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
	type Plain ImageContent
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ImageContent(plain)
	return nil
}

func (j ImageContent) CallToolResultContentType() ContentType {
	return ImageContentType
}

func (j ImageContent) PromptMessageContentType() ContentType {
	return ImageContentType
}

func (j ImageContent) SamplingMessageContentType() ContentType {
	return ImageContentType
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
	type Plain Implementation
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Implementation(plain)
	return nil
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
	type Plain InitializeRequestParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = InitializeRequestParams(plain)
	return nil
}

// This request is sent from the client to the server when it first connects,
// asking it to begin initialization.
type InitializeRequest struct {
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params InitializeRequestParams `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *InitializeRequest) UnmarshalJSON(b []byte) error {
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
	type Plain InitializeRequest
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = InitializeRequest(plain)
	return nil
}

// After receiving an initialize request from the client, the server sends this
// response.
type InitializeResult struct {
	// This result property is reserved by the protocol to allow clients and servers
	// to attach additional metadata to their responses.
	Meta InitializeResultMeta `json:"_meta,omitempty"`
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

// This result property is reserved by the protocol to allow clients and servers to
// attach additional metadata to their responses.
type InitializeResultMeta map[string]any

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
	type Plain InitializeResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = InitializeResult(plain)
	return nil
}

type InitializedNotificationParams struct {
	// This parameter name is reserved by MCP to allow clients and servers to attach
	// additional metadata to their notifications.
	Meta InitializedNotificationParamsMeta `json:"_meta,omitempty"`
	// AdditionalProperties reserved for future use.
	AdditionalProperties any `json:",omitempty"`
}

// This parameter name is reserved by MCP to allow clients and servers to attach
// additional metadata to their notifications.
type InitializedNotificationParamsMeta map[string]any

// This notification is sent from the client to the server after initialization has
// finished.
type InitializedNotification struct {
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params *InitializedNotificationParams `json:"params,omitempty"`
}

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
	type Plain InitializedNotification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = InitializedNotification(plain)
	return nil
}

// Sent from the client to request a list of prompts and prompt templates the
// server has.
type ListPromptsRequest struct {
	PaginatedRequest
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListPromptsRequest) UnmarshalJSON(b []byte) error {
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
	type Plain ListPromptsRequest
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListPromptsRequest(plain)
	return nil
}

// The server's response to a prompts/list request from the client.
type ListPromptsResult struct {
	PaginatedResult
	// Prompts corresponds to the JSON schema field "prompts".
	Prompts []Prompt `json:"prompts"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListPromptsResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["prompts"]; raw != nil && !ok {
		return fmt.Errorf("field prompts in ListPromptsResult: required")
	}
	type Plain ListPromptsResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListPromptsResult(plain)
	return nil
}

// Sent from the client to request a list of resource templates the server has.
type ListResourceTemplatesRequest struct {
	PaginatedRequest
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListResourceTemplatesRequest) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["method"]
	if raw != nil && !ok {
		return fmt.Errorf("field method in ListResourceTemplatesRequest: required")
	}
	if strVal, ok := val.(string); !ok || RequestMethod(strVal) != ListResourceTemplRequestMethod {
		return fmt.Errorf("invalid field method in ListResourceTemplatesRequest: %v", strVal)
	}
	type Plain ListResourceTemplatesRequest
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListResourceTemplatesRequest(plain)
	return nil
}

// The server's response to a resources/templates/list request from the client.
type ListResourceTemplatesResult struct {
	PaginatedResult
	// ResourceTemplates corresponds to the JSON schema field "resourceTemplates".
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListResourceTemplatesResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["resourceTemplates"]; raw != nil && !ok {
		return fmt.Errorf("field resourceTemplates in ListResourceTemplatesResult: required")
	}
	type Plain ListResourceTemplatesResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListResourceTemplatesResult(plain)
	return nil
}

// Sent from the client to request a list of resources the server has.
type ListResourcesRequest struct {
	PaginatedRequest
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListResourcesRequest) UnmarshalJSON(b []byte) error {
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
	type Plain ListResourcesRequest
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListResourcesRequest(plain)
	return nil
}

// The server's response to a resources/list request from the client.
type ListResourcesResult struct {
	PaginatedResult
	// Resources corresponds to the JSON schema field "resources".
	Resources []Resource `json:"resources"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListResourcesResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["resources"]; raw != nil && !ok {
		return fmt.Errorf("field resources in ListResourcesResult: required")
	}
	type Plain ListResourcesResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListResourcesResult(plain)
	return nil
}

// Sent from the server to request a list of root URIs from the client. Roots allow
// servers to ask for specific directories or files to operate on. A common example
// for roots is providing a set of repositories or directories a server should
// operate
// on.
//
// This request is typically used when the server needs to understand the file
// system
// structure or access specific locations that the client has permission to read
// from.
type ListRootsRequest[T Token] struct {
	Request[T]
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListRootsRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain ListRootsRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListRootsRequest[T](plain)
	return nil
}

// The client's response to a roots/list request from the server.
// This result contains an array of Root objects, each representing a root
// directory
// or file that the server can operate on.
type ListRootsResult struct {
	Result
	// Roots corresponds to the JSON schema field "roots".
	Roots []Root `json:"roots"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListRootsResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["roots"]; raw != nil && !ok {
		return fmt.Errorf("field roots in ListRootsResult: required")
	}
	type Plain ListRootsResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListRootsResult(plain)
	return nil
}

// Sent from the client to request a list of tools the server has.
type ListToolsRequest struct {
	PaginatedRequest
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListToolsRequest) UnmarshalJSON(b []byte) error {
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
	type Plain ListToolsRequest
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListToolsRequest(plain)
	return nil
}

// The server's response to a tools/list request from the client.
type ListToolsResult struct {
	PaginatedResult
	// Tools corresponds to the JSON schema field "tools".
	Tools []Tool `json:"tools"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ListToolsResult) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["tools"]; raw != nil && !ok {
		return fmt.Errorf("field tools in ListToolsResult: required")
	}
	type Plain ListToolsResult
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ListToolsResult(plain)
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

var enumValuesLoggingLevel = map[LoggingLevel]struct{}{
	LoggingLevelAlert:     {},
	LoggingLevelCritical:  {},
	LoggingLevelDebug:     {},
	LoggingLevelEmergency: {},
	LoggingLevelError:     {},
	LoggingLevelInfo:      {},
	LoggingLevelNotice:    {},
	LoggingLevelWarning:   {},
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *LoggingLevel) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	level := LoggingLevel(v)
	if _, valid := enumValuesLoggingLevel[level]; !valid {
		return fmt.Errorf("invalid logginglevel value: %v", level)
	}

	*j = level
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
func (j *LoggingMessageNotificationParams) UnmarshalJSON(b []byte) error {
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
	type Plain LoggingMessageNotificationParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = LoggingMessageNotificationParams(plain)
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

// UnmarshalJSON implements json.Unmarshaler.
func (j *LoggingMessageNotification) UnmarshalJSON(b []byte) error {
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
	type Plain LoggingMessageNotification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = LoggingMessageNotification(plain)
	return nil
}

// Hints to use for model selection.
//
// Keys not declared here are currently left unspecified by the spec and are up
// to the client to interpret.
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
func (j *ModelPreferences) UnmarshalJSON(b []byte) error {
	type Plain ModelPreferences
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	if plain.CostPriority != nil && 1 < *plain.CostPriority {
		return fmt.Errorf("field %s: must be <= %v", "costPriority", 1)
	}
	if plain.CostPriority != nil && 0 > *plain.CostPriority {
		return fmt.Errorf("field %s: must be >= %v", "costPriority", 0)
	}
	if plain.IntelligencePriority != nil && 1 < *plain.IntelligencePriority {
		return fmt.Errorf("field %s: must be <= %v", "intelligencePriority", 1)
	}
	if plain.IntelligencePriority != nil && 0 > *plain.IntelligencePriority {
		return fmt.Errorf("field %s: must be >= %v", "intelligencePriority", 0)
	}
	if plain.SpeedPriority != nil && 1 < *plain.SpeedPriority {
		return fmt.Errorf("field %s: must be <= %v", "speedPriority", 1)
	}
	if plain.SpeedPriority != nil && 0 > *plain.SpeedPriority {
		return fmt.Errorf("field %s: must be >= %v", "speedPriority", 0)
	}
	*j = ModelPreferences(plain)
	return nil
}

// This parameter name is reserved by MCP to allow clients and servers to attach
// additional metadata to their notifications.
type NotificationParamsMeta map[string]any

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

// UnmarshalJSON implements json.Unmarshaler.
func (j *Notification) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["method"]; raw != nil && !ok {
		return fmt.Errorf("field method in Notification: required")
	}
	type Plain Notification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Notification(plain)
	return nil
}

type PaginatedRequestParams struct {
	// An opaque token representing the current pagination position.
	// If provided, the server should return results starting after this cursor.
	Cursor *Cursor `json:"cursor,omitempty"`
}

type PaginatedRequest struct {
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params *PaginatedRequestParams `json:"params,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PaginatedRequest) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["method"]; raw != nil && !ok {
		return fmt.Errorf("field method in PaginatedRequest: required")
	}
	type Plain PaginatedRequest
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PaginatedRequest(plain)
	return nil
}

type PaginatedResult struct {
	// This result property is reserved by the protocol to allow clients and servers
	// to attach additional metadata to their responses.
	Meta PaginatedResultMeta `json:"_meta,omitempty"`
	// An opaque token representing the pagination position after the last returned
	// result.
	// If present, there may be more results available.
	NextCursor *Cursor `json:"nextCursor,omitempty"`
}

// This result property is reserved by the protocol to allow clients and servers to
// attach additional metadata to their responses.
type PaginatedResultMeta map[string]any

type PingRequestParams[T Token] struct {
	// Meta corresponds to the JSON schema field "_meta".
	Meta *PingRequestParamsMeta[T] `json:"_meta,omitempty"`
	// AdditionalProperties reserved for future use.
	AdditionalProperties any `json:",omitempty"`
}

type PingRequestParamsMeta[T Token] struct {
	// If specified, the caller is requesting out-of-band progress notifications for
	// this request (as represented by notifications/progress). The value of this
	// parameter is an opaque token that will be attached to any subsequent
	// notifications. The receiver is not obligated to provide these notifications.
	ProgressToken *ProgressToken[T] `json:"progressToken,omitempty"`
}

// A ping, issued by either the server or the client, to check that the other party
// is still alive. The receiver must promptly respond, or else may be disconnected.
type PingRequest[T Token] struct {
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params *PingRequestParams[T] `json:"params,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PingRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain PingRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PingRequest[T](plain)
	return nil
}

type ProgressNotificationParams[T Token] struct {
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
func (j *ProgressNotificationParams[T]) UnmarshalJSON(b []byte) error {
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
	type Plain ProgressNotificationParams[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ProgressNotificationParams[T](plain)
	return nil
}

// An out-of-band notification used to inform the receiver of a progress update for
// a long-running request.
type ProgressNotification[T Token] struct {
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params ProgressNotificationParams[T] `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ProgressNotification[T]) UnmarshalJSON(b []byte) error {
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
	type Plain ProgressNotification[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ProgressNotification[T](plain)
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
func (j *PromptArgument) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in PromptArgument: required")
	}
	type Plain PromptArgument
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PromptArgument(plain)
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
func (j *Prompt) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in Prompt: required")
	}
	type Plain Prompt
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Prompt(plain)
	return nil
}

// An optional notification from the server to the client, informing it that the
// list of prompts it offers has changed. This may be issued by servers without any
// previous subscription from the client.
type PromptListChangedNotification struct {
	Notification
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *PromptListChangedNotification) UnmarshalJSON(b []byte) error {
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
	type Plain PromptListChangedNotification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PromptListChangedNotification(plain)
	return nil
}

type PromptMessageContent interface {
	PromptMessageContentType() ContentType
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
	type Alias PromptMessage
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	if aux.Content == nil {
		return fmt.Errorf("field content in PromptMessage: required")
	}

	if _, valid := enumValuesRole[aux.Role]; !valid {
		return fmt.Errorf("invalid role value: %v", aux.Role)
	}
	j.Role = aux.Role

	// Try each possible content type
	var text TextContent
	if err := json.Unmarshal(aux.Content, &text); err == nil {
		j.Content = &text
		return nil
	}

	var image ImageContent
	if err := json.Unmarshal(aux.Content, &image); err == nil {
		j.Content = &image
		return nil
	}

	var embedded EmbeddedResource
	if err := json.Unmarshal(aux.Content, &embedded); err == nil {
		j.Content = &embedded
		return nil
	}

	return fmt.Errorf("content matches neither TextContent, ImageContent, nor EmbeddedResource")
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
	type Plain PromptReference
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = PromptReference(plain)
	return nil
}

func (j PromptReference) CompleteRequestParamsRefType() ReferenceType {
	return j.Type
}

type ReadResourceRequestParams struct {
	// The URI of the resource to read. The URI can use any protocol; it is up to the
	// server how to interpret it.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ReadResourceRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in ReadResourceRequestParams: required")
	}
	type Plain ReadResourceRequestParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ReadResourceRequestParams(plain)
	return nil
}

// Sent from the client to the server, to read a specific resource URI.
type ReadResourceRequest[T Token] struct {
	Request[T]
	Params *ReadResourceRequestParams `json:"params,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ReadResourceRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain ReadResourceRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ReadResourceRequest[T](plain)
	return nil
}

type ReadResourceResultContent interface {
	ReadResourceResultContentType() ResourceContentType
}

// The server's response to a resources/read request from the client.
type ReadResourceResult struct {
	Result
	// Contents corresponds to the JSON schema field "contents".
	// TextResourceContents | BlobResourceContents
	Contents []ReadResourceResultContent `json:"contents"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ReadResourceResult) UnmarshalJSON(b []byte) error {
	type Alias ReadResourceResult
	aux := &struct {
		Contents []json.RawMessage `json:"contents"`
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	if aux.Contents == nil {
		return fmt.Errorf("field contents in ReadResourceResult: required")
	}

	j.Contents = make([]ReadResourceResultContent, len(aux.Contents))
	for i, raw := range aux.Contents {
		var text TextResourceContents
		if err := json.Unmarshal(raw, &text); err == nil {
			j.Contents[i] = &text
			continue
		}

		var blob BlobResourceContents
		if err := json.Unmarshal(raw, &blob); err == nil {
			j.Contents[i] = &blob
			continue
		}

		return fmt.Errorf("content at index %d matches neither type", i)
	}

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
	ListResourceTemplRequestMethod        RequestMethod = "resources/templates/list"
	SubscribeRequestMethod                RequestMethod = "resources/subscribe"
	UnsubscribeRequestMethod              RequestMethod = "resources/unsubscribe"
	ReadResourceRequestMethod             RequestMethod = "resources/read"
	ListRootsRequestMethod                RequestMethod = "roots/list"
	CreateMessageRequestMethod            RequestMethod = "sampling/createMessage"
	ListToolsRequestMethod                RequestMethod = "tools/list"
	CallToolRequestMethod                 RequestMethod = "tools/call"
)

// ID is used as a generic constraint.
// NOTE: this is mostly to make the semantics clearer
// to the code readers; the interface could be extended, too.
type ID StringOrNumber

// RequestID is a uniquely identifying ID for a request in JSON-RPC.
// NOTE we could also define a struct like this:
//
//	type UnionValue[T StringOrNumber] struct {
//	    Value T
//	}
//
// # And then then ProgressToken like so
//
// type RequestID = UnionValue[StringOrNumber]
type RequestID[T StringOrNumber] struct {
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
		return fmt.Errorf("failed to unmarshal union value: %w", err)
	}
	r.Value = v
	return nil
}

type RequestParams[T Token] struct {
	// Meta corresponds to the JSON schema field "_meta".
	Meta *RequestParamsMeta[T] `json:"_meta,omitempty"`
	// AdditionalProperties are reserved for future use.
	AdditionalProperties any `json:",omitempty"`
}

type RequestParamsMeta[T Token] struct {
	// If specified, the caller is requesting out-of-band progress notifications for
	// this request (as represented by notifications/progress). The value of this
	// parameter is an opaque token that will be attached to any subsequent
	// notifications. The receiver is not obligated to provide these notifications.
	ProgressToken *ProgressToken[T] `json:"progressToken,omitempty"`
}

type Request[T Token] struct {
	// Method corresponds to the JSON schema field "method".
	Method RequestMethod `json:"method"`
	// Params corresponds to the JSON schema field "params".
	Params *RequestParams[T] `json:"params,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Request[T]) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["method"]; raw != nil && !ok {
		return fmt.Errorf("field method in Request: required")
	}
	type Plain Request[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Request[T](plain)
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

// The contents of a specific resource or sub-resource.
type ResourceContents struct {
	// The MIME type of this resource, if known.
	MimeType *string `json:"mimeType,omitempty"`
	// The URI of this resource.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ResourceContents) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in ResourceContents: required")
	}
	type Plain ResourceContents
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ResourceContents(plain)
	return nil
}

// An optional notification from the server to the client, informing it that the
// list of resources it can read from has changed. This may be issued by servers
// without any previous subscription from the client.
type ResourceListChangedNotification struct {
	Notification
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ResourceListChangedNotification) UnmarshalJSON(b []byte) error {
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
	if _, ok := raw["params"]; raw != nil && !ok {
		return fmt.Errorf("field params in ReadResourceRequest: required")
	}
	type Plain ResourceListChangedNotification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ResourceListChangedNotification(plain)
	return nil
}

// A reference to a resource or resource template definition.
type ResourceReference struct {
	// Type corresponds to the JSON schema field "type".
	Type ReferenceType `json:"type"`
	// The URI or URI template of the resource.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ResourceReference) UnmarshalJSON(b []byte) error {
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
	type Plain ResourceReference
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ResourceReference(plain)
	return nil
}

func (j ResourceReference) CompleteRequestParamsRefType() ReferenceType {
	return j.Type
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
func (j *ResourceTemplate) UnmarshalJSON(b []byte) error {
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
	type Plain ResourceTemplate
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ResourceTemplate(plain)
	return nil
}

type ResourceUpdatedNotificationParams struct {
	// The URI of the resource that has been updated. This might be a sub-resource of
	// the one that the client actually subscribed to.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ResourceUpdatedNotificationParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in ResourceUpdatedNotificationParams: required")
	}
	type Plain ResourceUpdatedNotificationParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ResourceUpdatedNotificationParams(plain)
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

// UnmarshalJSON implements json.Unmarshaler.
func (j *ResourceUpdatedNotification) UnmarshalJSON(b []byte) error {
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
	type Plain ResourceUpdatedNotification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ResourceUpdatedNotification(plain)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *Resource) UnmarshalJSON(b []byte) error {
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
	type Plain Resource
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Resource(plain)
	return nil
}

// This result property is reserved by the protocol to allow clients and servers to
// attach additional metadata to their responses.
type ResultMeta map[string]any

type Result struct {
	// This result property is reserved by the protocol to allow clients and servers
	// to attach additional metadata to their responses.
	Meta ResultMeta `json:"_meta,omitempty"`
	// AdditionalProperties are reserved for future use.
	AdditionalProperties any `json:",omitempty"`
}

type Role string

const (
	RoleAssistant Role = "assistant"
	RoleUser      Role = "user"
)

var enumValuesRole = map[Role]struct{}{
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
	if _, valid := enumValuesRole[role]; !valid {
		return fmt.Errorf("invalid role value: %v", s)
	}

	*r = role
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
func (j *Root) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in Root: required")
	}
	type Plain Root
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Root(plain)
	return nil
}

// A notification from the client to the server, informing it that the list of
// roots has changed.
// This notification should be sent whenever the client adds, removes, or modifies
// any root.
// The server should then request an updated list of roots using the
// ListRootsRequest.
type RootsListChangedNotification struct {
	Notification
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *RootsListChangedNotification) UnmarshalJSON(b []byte) error {
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
	type Plain RootsListChangedNotification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = RootsListChangedNotification(plain)
	return nil
}

type SamplingMessageContent interface {
	SamplingMessageContentType() ContentType
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
	type Alias SamplingMessage
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	if aux.Content == nil {
		return fmt.Errorf("field content in SamplingMessage: required")
	}

	if _, valid := enumValuesRole[aux.Role]; !valid {
		return fmt.Errorf("invalid role value: %v", aux.Role)
	}
	j.Role = aux.Role

	// Try each possible content type
	var text TextContent
	if err := json.Unmarshal(aux.Content, &text); err == nil {
		j.Content = &text
		return nil
	}

	var image ImageContent
	if err := json.Unmarshal(aux.Content, &image); err == nil {
		j.Content = &image
		return nil
	}

	return fmt.Errorf("content matches neither TextContent nor ImageContent")
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

type SetLevelRequestParams struct {
	// The level of logging that the client wants to receive from the server. The
	// server should send all logs at this level and higher (i.e., more severe) to the
	// client as notifications/logging/message.
	Level LoggingLevel `json:"level"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SetLevelRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["level"]; raw != nil && !ok {
		return fmt.Errorf("field level in SetLevelRequestParams: required")
	}
	type Plain SetLevelRequestParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SetLevelRequestParams(plain)
	return nil
}

// A request from the client to the server, to enable or adjust logging.
type SetLevelRequest[T Token] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params SetLevelRequestParams `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SetLevelRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain SetLevelRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SetLevelRequest[T](plain)
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
	type Plain SubscribeRequestParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SubscribeRequestParams(plain)
	return nil
}

// Sent from the client to request resources/updated notifications from the server
// whenever a particular resource changes.
type SubscribeRequest[T Token] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params SubscribeRequestParams `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SubscribeRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain SubscribeRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = SubscribeRequest[T](plain)
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
	type Plain TextContent
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = TextContent(plain)
	return nil
}

func (j TextContent) CallToolResultContentType() ContentType {
	return TextContentType
}

func (j TextContent) PromptMessageContentType() ContentType {
	return TextContentType
}

func (j TextContent) SamplingMessageContentType() ContentType {
	return TextContentType
}

type TextResourceContents struct {
	ResourceContents
	// The text of the item. This must only be set if the item can actually be
	// represented as text (not binary data).
	Text string `json:"text"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *TextResourceContents) UnmarshalJSON(b []byte) error {
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
	type Plain TextResourceContents
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = TextResourceContents(plain)
	return nil
}

func (j TextResourceContents) EmbeddedResourceContentType() ResourceContentType {
	return TextResourceContentsType
}

func (j TextResourceContents) ReadResourceResultContentType() ResourceContentType {
	return TextResourceContentsType
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
	type Plain BlobResourceContents
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = BlobResourceContents(plain)
	return nil
}

func (j BlobResourceContents) EmbeddedResourceContentType() ResourceContentType {
	return BlobResourceContentsType
}

func (j BlobResourceContents) ReadResourceResultContentType() ResourceContentType {
	return BlobResourceContentsType
}

// A JSON Schema object defining the expected parameters for the tool.
type ToolInputSchema struct {
	// Type corresponds to the JSON schema field "type".
	Type ContentType `json:"type"`
	// Properties corresponds to the JSON schema field "properties".
	Properties ToolInputSchemaProperties `json:"properties,omitempty"`
}

type ToolInputSchemaProperties map[string]map[string]any

// UnmarshalJSON implements json.Unmarshaler.
func (j *ToolInputSchema) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	val, ok := raw["type"]
	if raw != nil && !ok {
		return fmt.Errorf("field type in ToolInputSchema: required")
	}
	if strVal, ok := val.(string); !ok || ContentType(strVal) != ObjectType {
		return fmt.Errorf("invalid field type in ToolInputSchema: %v", strVal)
	}
	type Plain ToolInputSchema
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ToolInputSchema(plain)
	return nil
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

// UnmarshalJSON implements json.Unmarshaler.
func (j *Tool) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["inputSchema"]; raw != nil && !ok {
		return fmt.Errorf("field inputSchema in Tool: required")
	}
	if _, ok := raw["name"]; raw != nil && !ok {
		return fmt.Errorf("field name in Tool: required")
	}
	type Plain Tool
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = Tool(plain)
	return nil
}

// An optional notification from the server to the client, informing it that the
// list of tools it offers has changed. This may be issued by servers without any
// previous subscription from the client.
type ToolListChangedNotification struct {
	Notification
}

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
	type Plain ToolListChangedNotification
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = ToolListChangedNotification(plain)
	return nil
}

type UnsubscribeRequestParams struct {
	// The URI of the resource to unsubscribe from.
	URI string `json:"uri"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *UnsubscribeRequestParams) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["uri"]; raw != nil && !ok {
		return fmt.Errorf("field uri in UnsubscribeRequestParams: required")
	}
	type Plain UnsubscribeRequestParams
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = UnsubscribeRequestParams(plain)
	return nil
}

// Sent from the client to request cancellation of resources/updated notifications
// from the server. This should follow a previous resources/subscribe request.
type UnsubscribeRequest[T Token] struct {
	Request[T]
	// Params corresponds to the JSON schema field "params".
	Params UnsubscribeRequestParams `json:"params"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *UnsubscribeRequest[T]) UnmarshalJSON(b []byte) error {
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
	type Plain UnsubscribeRequest[T]
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = UnsubscribeRequest[T](plain)
	return nil
}
