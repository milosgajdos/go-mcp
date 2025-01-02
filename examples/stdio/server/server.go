// server/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/milosgajdos/go-mcp"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("server: ")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("starting up...")

	transport := mcp.NewStdioTransport[uint64]()

	server, err := mcp.NewServer[uint64](
		mcp.WithServerTransport(transport),
		mcp.WithServerCapabilities(mcp.ServerCapabilities{
			Tools:     &mcp.ServerCapabilitiesTools{},
			Resources: &mcp.ServerCapabilitiesResources{},
			Prompts:   &mcp.ServerCapabilitiesPrompts{},
		}),
	)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := server.Connect(ctx); err != nil {
		log.Fatalf("failed to connect server: %v", err)
	}
	log.Printf("server connected and ready - waiting for requests")

	<-sigChan
	log.Printf("shutting down...")
	if err := server.Close(ctx); err != nil {
		log.Printf("error closing server: %v", err)
	}
	log.Println("shut down")
}
