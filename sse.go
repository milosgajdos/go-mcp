package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

const (
	// DefaultMaxMessageSize sets the maximum size of incoming messages
	DefaultMaxMessageSize = 4 * 1024 * 1024 // 4MB
)

// Ensure SSETransport implements Transport interface
var _ Transport = (*SSETransport)(nil)

// SSETransport implements Transport interface using Server-Sent Events
type SSETransport struct {
	options TransportOptions

	// HTTP components
	server   *http.Server
	client   *http.Client
	isServer bool

	// Server URLs
	serverURL string
	postURL   string

	// Message channels
	outgoing chan JSONRPCMessage
	incoming chan JSONRPCMessage
	done     chan struct{}

	// WaitGroup to track background goroutines
	wg sync.WaitGroup

	// Session management
	sessionID     string
	activeSession sync.Map // maps sessionID to writer

	state atomic.Int32
}

// NewSSETransport initializes a new SSE transport
func NewSSETransport(serverURL string, opts ...TransportOption) (*SSETransport, error) {
	options := TransportOptions{}
	for _, apply := range opts {
		apply(&options)
	}

	// Validate URL
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// If no scheme is provided, assume we're in server mode
	isServer := u.Scheme == ""

	t := &SSETransport{
		options:   options,
		serverURL: serverURL,
		client:    &http.Client{},
		isServer:  isServer,
	}

	if isServer {
		mux := http.NewServeMux()
		mux.HandleFunc("/sse", t.handleSSE)
		mux.HandleFunc("/message", t.handleMessage)

		t.server = &http.Server{
			Addr:    serverURL,
			Handler: mux,
		}
	}

	return t, nil
}

// Start initializes the transport
func (t *SSETransport) Start(ctx context.Context) error {
	if !t.state.CompareAndSwap(int32(stateStopped), int32(stateRunning)) {
		return ErrTransportStarted
	}

	t.outgoing = make(chan JSONRPCMessage, 100)
	t.incoming = make(chan JSONRPCMessage, 100)
	t.done = make(chan struct{})

	// Generate session ID
	t.sessionID = uuid.NewString()

	if t.isServer {
		// Start server
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			if err := t.server.ListenAndServe(); err != http.ErrServerClosed {
				fmt.Printf("server error: %v\n", err)
			}
		}()
		return nil
	}

	// Client mode: establish SSE connection
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.handleClientSSE(ctx)
	}()

	return nil
}

func (t *SSETransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Get or generate session ID
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send endpoint URL with session ID
	fmt.Fprintf(w, "event: endpoint\ndata: /message?sessionId=%s\n\n", sessionID)
	flusher.Flush()

	// Store writer for this session
	t.activeSession.Store(sessionID, w)
	defer t.activeSession.Delete(sessionID)

	// Forward messages to this client
	for {
		select {
		case msg := <-t.outgoing:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		case <-t.done:
			return
		}
	}
}

func (t *SSETransport) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	if _, ok := t.activeSession.Load(sessionID); !ok {
		http.Error(w, "Invalid session ID", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, DefaultMaxMessageSize))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var msg JSONRPCMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, "Invalid JSON message", http.StatusBadRequest)
		return
	}

	select {
	case t.incoming <- msg:
		w.WriteHeader(http.StatusAccepted)
	case <-t.done:
		http.Error(w, "Transport closed", http.StatusServiceUnavailable)
	default:
		http.Error(w, "Message queue full", http.StatusServiceUnavailable)
	}
}

// Send sends a message
func (t *SSETransport) Send(ctx context.Context, msg JSONRPCMessage) error {
	if t.state.Load() != int32(stateRunning) {
		return ErrTransportClosed
	}

	if t.isServer {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t.outgoing <- msg:
			return nil
		}
	}

	// Client mode: send via POST request
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	postURL := fmt.Sprintf("%s?sessionId=%s", t.postURL, t.sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response: %s (%d)", string(body), resp.StatusCode)
	}

	return nil
}

// Receive receives a message
func (t *SSETransport) Receive(ctx context.Context) (JSONRPCMessage, error) {
	if t.state.Load() != int32(stateRunning) {
		return nil, ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-t.incoming:
		return msg, nil
	}
}

// handleClientSSE manages the client-side SSE connection
func (t *SSETransport) handleClientSSE(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.serverURL+"/sse", nil)
	if err != nil {
		fmt.Printf("create SSE request failed: %v\n", err)
		return
	}

	resp, err := t.client.Do(req)
	if err != nil {
		fmt.Printf("SSE connection failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Printf("read SSE event failed: %v\n", err)
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			data, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("read SSE data failed: %v\n", err)
				return
			}
			data = strings.TrimPrefix(strings.TrimSpace(data), "data: ")

			switch eventType {
			case "endpoint":
				postURL, err := url.Parse(data)
				if err != nil {
					fmt.Printf("invalid endpoint URL: %v\n", err)
					return
				}
				t.postURL = postURL.String()
			case "message":
				var msg JSONRPCMessage
				if err := json.Unmarshal([]byte(data), &msg); err != nil {
					fmt.Printf("invalid message data: %v\n", err)
					continue
				}
				select {
				case t.incoming <- msg:
				case <-t.done:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// Close terminates the transport
func (t *SSETransport) Close() error {
	if !t.state.CompareAndSwap(int32(stateRunning), int32(stateStopped)) {
		return nil
	}

	close(t.done)

	if t.server != nil {
		if err := t.server.Close(); err != nil {
			return fmt.Errorf("close server: %w", err)
		}
	}

	t.wg.Wait()

	close(t.incoming)
	close(t.outgoing)

	return nil
}
