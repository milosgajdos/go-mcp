package mcp

import "context"

type Transport interface {
	// Send transmits data to the remote endpoint
	Send(ctx context.Context, data []byte) error
	// Receive waits for and returns the next message from the remote endpoint
	Receive(ctx context.Context) ([]byte, error)
	// Close terminates the transport connection and cleans up resources
	Close() error
}
