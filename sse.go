package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
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

// SSEServerTransport implements Transport interface for server-side SSE
type SSEServerTransport[T ID] struct {
	options TransportOptions
	server  *http.Server
	addr    string

	sessionID string
	incoming  chan JSONRPCMessage
	outgoing  chan JSONRPCMessage
	done      chan struct{}

	mu    sync.Mutex // protects server startup/shutdown
	wg    sync.WaitGroup
	state atomic.Int32
}

func NewSSEServerTransport[T ID](addr string, opts ...TransportOption) (*SSEServerTransport[T], error) {
	options := TransportOptions{}
	for _, apply := range opts {
		apply(&options)
	}

	return &SSEServerTransport[T]{
		options: options,
		addr:    addr,
	}, nil
}

func (s *SSEServerTransport[T]) Start(context.Context) error {
	if !s.state.CompareAndSwap(int32(stateStopped), int32(stateRunning)) {
		return ErrTransportStarted
	}

	// prevents race conditions when closing
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionID = uuid.NewString()

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)

	// Create the listener on :0 so we can find the actual ephemeral port
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.addr, err)
	}

	// let's store that
	s.server = &http.Server{
		Handler: mux,
	}
	// Make sure we update s.addr to the actual ephemeral port
	s.addr = ln.Addr().String()

	s.incoming = make(chan JSONRPCMessage, 100)
	s.outgoing = make(chan JSONRPCMessage, 100)
	s.done = make(chan struct{})

	ready := make(chan struct{})

	// captured the locked server
	server := s.server

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		close(ready)
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v\n", err)
		}
	}()

	<-ready
	return nil
}

func (s *SSEServerTransport[T]) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send session ID
	fmt.Fprintf(w, "event: endpoint\ndata: /message?sessionId=%s\n\n", s.sessionID)
	flusher.Flush()

	// Forward messages to client
	for {
		select {
		case msg := <-s.outgoing:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		case <-s.done:
			return
		}
	}
}

func (s *SSEServerTransport[T]) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID != s.sessionID {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, DefaultMaxMessageSize))
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var msg JSONRPCMessage
	msg, err = parseJSONRPCMessage[T](body)
	if err != nil {
		http.Error(w, "Invalid message", http.StatusBadRequest)
		return
	}

	select {
	case s.incoming <- msg:
		w.WriteHeader(http.StatusAccepted)
	case <-r.Context().Done():
		return
	case <-s.done:
		return
	default:
		http.Error(w, "Queue full", http.StatusServiceUnavailable)
	}
}

func (s *SSEServerTransport[T]) Send(ctx context.Context, msg JSONRPCMessage) error {
	if s.state.Load() != int32(stateRunning) {
		return ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return ErrTransportClosed
	case s.outgoing <- msg:
		return nil
	}
}

func (s *SSEServerTransport[T]) Receive(ctx context.Context) (JSONRPCMessage, error) {
	if s.state.Load() != int32(stateRunning) {
		return nil, ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.done:
		return nil, ErrTransportClosed
	case msg := <-s.incoming:
		return msg, nil
	}
}

func (s *SSEServerTransport[T]) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

func (s *SSEServerTransport[T]) Close() error {
	if !s.state.CompareAndSwap(int32(stateRunning), int32(stateStopped)) {
		return nil
	}

	close(s.done)

	s.mu.Lock()
	if s.server != nil {
		if err := s.server.Close(); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("server close: %w", err)
		}
		s.server = nil
	}
	s.mu.Unlock()

	s.wg.Wait()
	close(s.incoming)
	close(s.outgoing)

	return nil
}

// SSEClientTransport implements Transport interface for client-side SSE
type SSEClientTransport[T ID] struct {
	options TransportOptions
	client  *http.Client
	url     string

	postURL  string
	incoming chan JSONRPCMessage
	done     chan struct{}

	wg    sync.WaitGroup
	state atomic.Int32
}

func NewSSEClientTransport[T ID](serverURL string, opts ...TransportOption) (*SSEClientTransport[T], error) {
	options := TransportOptions{}
	for _, apply := range opts {
		apply(&options)
	}

	_, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	return &SSEClientTransport[T]{
		options:  options,
		client:   &http.Client{},
		url:      serverURL,
		incoming: make(chan JSONRPCMessage, 100),
		done:     make(chan struct{}),
	}, nil
}

func (c *SSEClientTransport[T]) Start(ctx context.Context) error {
	if !c.state.CompareAndSwap(int32(stateStopped), int32(stateRunning)) {
		return ErrTransportStarted
	}

	started := make(chan error, 1)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.handleClientSSE(ctx, started)
	}()

	if err := <-started; err != nil {
		return err
	}

	return nil
}

// handleClientSSE manages the client-side SSE connection
func (c *SSEClientTransport[T]) handleClientSSE(ctx context.Context, errCh chan error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/sse", nil)
	if err != nil {
		errCh <- fmt.Errorf("create SSE request failed: %v", err)
		return
	}

	resp, err := c.client.Do(req)
	if err != nil {
		errCh <- fmt.Errorf("SSE connection failed: %v", err)
		return
	}
	defer resp.Body.Close()

	errCh <- nil

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF || c.state.Load() == int32(stateStopped) {
				select {
				case c.incoming <- &JSONRPCError[T]{
					Version: JSONRPCVersion,
					Err: Error{
						Code:    JSONRPCConnectionClosed,
						Message: ErrTransportClosed.Error(),
					},
				}:
				case <-c.done:
				case <-ctx.Done():
				}
				return
			}
			log.Printf("read error: %v", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			var msg JSONRPCMessage
			eventType := strings.TrimPrefix(line, "event: ")
			data, err := reader.ReadString('\n')
			if err != nil {
				select {
				case c.incoming <- &JSONRPCError[T]{
					Version: JSONRPCVersion,
					Err: Error{
						Code:    JSONRPCConnectionClosed,
						Message: ErrTransportClosed.Error(),
					},
				}:
				case <-c.done:
				case <-ctx.Done():
				}
				return
			}

			data = strings.TrimPrefix(strings.TrimSpace(data), "data: ")

			switch eventType {
			case "endpoint":
				postURL, err := url.Parse(data)
				if err != nil {
					select {
					case c.incoming <- &JSONRPCError[T]{
						Version: JSONRPCVersion,
						Err: Error{
							Code:    JSONRPCConnectionClosed,
							Message: err.Error(),
						},
					}:
					case <-c.done:
					case <-ctx.Done():
					}
					return
				}
				c.postURL = c.url + postURL.String()

			case "message":
				msg, err = parseJSONRPCMessage[T]([]byte(data))
				if err != nil {
					msg = &JSONRPCError[T]{
						Version: JSONRPCVersion,
						Err: Error{
							Code:    JSONRPCParseError,
							Message: err.Error(),
						},
					}
				}
				select {
				case c.incoming <- msg:
				case <-c.done:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (c *SSEClientTransport[T]) Send(ctx context.Context, msg JSONRPCMessage) error {
	if c.state.Load() != int32(stateRunning) {
		return ErrTransportClosed
	}

	if c.postURL == "" {
		return fmt.Errorf("no endpoint URL")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.postURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func (c *SSEClientTransport[T]) Receive(ctx context.Context) (JSONRPCMessage, error) {
	if c.state.Load() != int32(stateRunning) {
		return nil, ErrTransportClosed
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrTransportClosed
	case msg := <-c.incoming:
		if msg.JSONRPCMessageType() == JSONRPCErrorMsgType {
			errMsg, ok := msg.(*JSONRPCError[T])
			if ok {
				if errMsg.Err.Code == JSONRPCConnectionClosed {
					if err := c.Close(); err != nil {
						return nil, fmt.Errorf("transport close: %v", err)
					}
					return nil, ErrTransportClosed
				}
			}
		}
		return msg, nil
	}
}

func (c *SSEClientTransport[T]) Close() error {
	if !c.state.CompareAndSwap(int32(stateRunning), int32(stateStopped)) {
		return nil
	}

	close(c.done)
	c.wg.Wait()
	close(c.incoming)

	return nil
}
