package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func SSEServerTransportMustStart(ctx context.Context, t *testing.T) (*SSEServerTransport, string) {
	srv, err := NewSSEServerTransport(":0") // Use dynamic port
	if err != nil {
		t.Fatalf("NewSSEServerTransport() error = %v", err)
	}
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return srv, "http://" + srv.Addr()
}

func SSEServerTransportMustClose(t *testing.T, tr *SSEServerTransport) {
	if err := tr.Close(); err != nil {
		t.Fatalf("failed closing transport: %v", err)
	}
}

func TestSSEServerTransport_Start(t *testing.T) {
	t.Run("successful start", func(t *testing.T) {
		srv, err := NewSSEServerTransport(":0")
		if err != nil {
			t.Fatalf("NewSSEServerTransport() error = %v", err)
		}
		if err := srv.Start(context.Background()); err != nil {
			t.Errorf("Start() error = %v", err)
		}
		SSEServerTransportMustClose(t, srv)
	})

	t.Run("double start", func(t *testing.T) {
		srv, err := NewSSEServerTransport(":0")
		if err != nil {
			t.Fatalf("NewSSEServerTransport() error = %v", err)
		}
		if err := srv.Start(context.Background()); err != nil {
			t.Fatalf("First Start() error = %v", err)
		}
		if err := srv.Start(context.Background()); !errors.Is(err, ErrTransportStarted) {
			t.Errorf("Second Start() error = %v, want %v", err, ErrTransportStarted)
		}
		SSEServerTransportMustClose(t, srv)
	})
}

func TestSSEServerTransport_Send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		ctx := context.Background()
		srv, addr := SSEServerTransportMustStart(ctx, t)
		defer SSEServerTransportMustClose(t, srv)

		readyCh := make(chan error, 1)
		msgCh := make(chan string, 1)

		go func() {
			// Connect to SSE
			resp, err := http.Get(addr + "/sse")
			if err != nil {
				readyCh <- fmt.Errorf("http.Get() error = %v", err)
				return
			}
			defer resp.Body.Close()

			scanner := bufio.NewScanner(resp.Body)
			gotEndpoint := false

			for scanner.Scan() {
				line := scanner.Text()
				switch {
				case strings.HasPrefix(line, "event: endpoint"):
					// We got the initial endpoint event
					gotEndpoint = true
					// signal the main goroutine: "we got the SSE connection
					readyCh <- nil
				case strings.HasPrefix(line, "event: message"):
					// Next line should be the `data:` line
					if !scanner.Scan() {
						msgCh <- "no data line"
						return
					}
					dataLine := scanner.Text()
					msgCh <- dataLine
					return
				case strings.HasPrefix(line, "error"):
					msgCh <- fmt.Sprintf("sse error: %v", line)
				}
			}
			// If we exit the loop, either EOF or error
			if err := scanner.Err(); err != nil {
				msgCh <- fmt.Sprintf("scanner error: %v", err)
			} else if !gotEndpoint {
				msgCh <- "never got endpoint event"
			}
		}()

		// Wait for SSE goroutine to get the endpoint
		if err := <-readyCh; err != nil {
			t.Fatalf("SSE connection error = %v", err)
		}

		// Now we can safely send from the server side
		msg := &JSONRPCRequest{
			Request: &PingRequest{
				Request: Request{
					Method: PingRequestMethod,
				},
			},
			ID:      NewRequestID(uint64(1)),
			Version: JSONRPCVersion,
		}
		if err := srv.Send(ctx, msg); err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		// Expect the "event: message" SSE with "data: ..." next
		dataLine := <-msgCh
		if !strings.HasPrefix(dataLine, "data: ") {
			t.Errorf("wanted data line, got = %q", dataLine)
		}
	})

	t.Run("send before start", func(t *testing.T) {
		srv, err := NewSSEServerTransport(":0")
		if err != nil {
			t.Fatalf("NewSSEServerTransport() error = %v", err)
		}
		if err := srv.Send(context.Background(), nil); !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Send() error = %v, want %v", err, ErrTransportClosed)
		}
	})
}

func TestSSEServerTransport_Receive(t *testing.T) {
	t.Run("successful receive", func(t *testing.T) {
		ctx := context.Background()
		srv, addr := SSEServerTransportMustStart(ctx, t)
		defer SSEServerTransportMustClose(t, srv)

		// Open the SSE stream to get the sessionId
		resp, err := http.Get(addr + "/sse")
		if err != nil {
			t.Fatalf("http.Get() error = %v", err)
		}
		defer resp.Body.Close()

		// Scan lines from the SSE connection looking for "data: /message?sessionId=..."
		scanner := bufio.NewScanner(resp.Body)
		var sessionID string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: /message?sessionId=") {
				sessionID = strings.TrimPrefix(line, "data: /message?sessionId=")
				break
			}
		}
		if err := scanner.Err(); err != nil {
			t.Fatalf("scanner error: %v", err)
		}
		if sessionID == "" {
			t.Fatalf("failed to parse sessionId from SSE stream")
		}

		// Craft and send a test JSON-RPC message
		msg := &JSONRPCRequest{
			Request: &PingRequest{
				Request: Request{
					Method: PingRequestMethod,
				},
			},
			ID:      NewRequestID(uint64(1)),
			Version: JSONRPCVersion,
		}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		urlString := addr + "/message?sessionId=" + sessionID
		req, err := http.NewRequest(http.MethodPost, urlString, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("http.NewRequest() error = %v", err)
		}

		respMsg, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /message error = %v", err)
		}
		defer respMsg.Body.Close()

		if respMsg.StatusCode != http.StatusAccepted {
			t.Errorf("status code = %v, want %v", respMsg.StatusCode, http.StatusAccepted)
		}

		// The SSEServerTransport should enqueue the message. Now we try to receive it:
		received, err := srv.Receive(ctx)
		if err != nil {
			t.Errorf("Receive() error = %v", err)
		}

		reqMsg, ok := received.(*JSONRPCRequest)
		if !ok {
			t.Fatalf("received type = %T, want *JSONRPCRequest", received)
		}

		if reqMsg.Version != JSONRPCVersion {
			t.Errorf("received JSON-RPC version = %q, want %q", reqMsg.Version, JSONRPCVersion)
		}
		if method := reqMsg.Request.GetMethod(); method != PingRequestMethod {
			t.Errorf("received method = %q, want %q", method, PingRequestMethod)
		}
	})

	t.Run("receive before start", func(t *testing.T) {
		srv, err := NewSSEServerTransport(":0")
		if err != nil {
			t.Fatalf("NewSSEServerTransport() error = %v", err)
		}

		// We haven't started srv, so receiving must fail with ErrTransportClosed
		if _, err := srv.Receive(context.Background()); !errors.Is(err, ErrTransportClosed) {
			t.Errorf("Receive() error = %v, want %v", err, ErrTransportClosed)
		}
	})
}

func TestSSEClientTransport(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		// Create a test server
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Send endpoint
			fmt.Fprintf(w, "event: endpoint\ndata: /message\n\n")
			flusher.Flush()

			// Send test message
			msg := &JSONRPCRequest{
				Request: &PingRequest{
					Request: Request{
						Method: PingRequestMethod,
					},
				},
				ID:      NewRequestID(uint64(1)),
				Version: JSONRPCVersion,
			}
			data, err := json.Marshal(msg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "marshal failed: %v\n", err)
				http.Error(w, "Marshal failed", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			flusher.Flush()
		}))
		defer srv.Close()

		client, err := NewSSEClientTransport(srv.URL)
		if err != nil {
			t.Fatalf("NewSSEClientTransport() error = %v", err)
		}

		ctx := context.Background()
		if err := client.Start(ctx); err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		msg, err := client.Receive(ctx)
		if err != nil {
			t.Fatalf("Receive() error = %v", err)
		}

		reqMsg, ok := msg.(*JSONRPCRequest)
		if !ok {
			t.Errorf("received message type = %T, want *JSONRPCRequest", msg)
		}
		if reqMsg.Version != JSONRPCVersion {
			t.Errorf("received message version = %v, want %v", reqMsg.Version, JSONRPCVersion)
		}

		if err := client.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}
