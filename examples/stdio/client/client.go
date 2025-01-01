// client/main.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/milosgajdos/go-mcp"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("client: ")
}

func main() {
	ctx := context.Background()

	log.Printf("starting up...")

	transport := mcp.NewStdioTransport()

	client, err := mcp.NewClient[uint64](
		mcp.WithClientTransport(transport),
		mcp.WithClientCapabilities(mcp.ClientCapabilities{
			Roots: &mcp.ClientCapabilitiesRoots{},
		}),
	)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("failed to connect client: %v", err)
	}
	log.Printf("connected to server")

	// Send ping using the public API
	log.Printf("sending ping request...")
	if err := client.Ping(ctx); err != nil {
		log.Printf("ping error: %v", err)
	} else {
		log.Printf("received ping response")
	}

	// List tools using the public API
	log.Printf("requesting tools...")
	tools, err := client.ListTools(ctx, nil)
	if err != nil {
		log.Printf("list tools error: %v", err)
	} else {
		log.Printf("received tools response: %+v", tools)
	}

	log.Printf("waiting for messages to be processed...")
	time.Sleep(time.Second)
	log.Printf("client finished")
}
