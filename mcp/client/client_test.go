// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNew(t *testing.T) {
	client := New("test-client", "v1.0.0", nil)
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.impl == nil {
		t.Fatal("expected non-nil underlying client")
	}
}

func TestNewWithOptions(t *testing.T) {
	opts := &Options{
		ClientOptions: &mcp.ClientOptions{},
	}

	client := New("test-client", "v1.0.0", opts)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestMCPClient(t *testing.T) {
	client := New("test-client", "v1.0.0", nil)

	impl := client.MCPClient()
	if impl == nil {
		t.Fatal("expected non-nil MCP client")
	}
}

func TestConnectWithInMemoryTransport(t *testing.T) {
	// Create a server and client using in-memory transports
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "v1.0.0",
	}, nil)

	// Add a simple tool to the server
	mcp.AddTool(server, &mcp.Tool{
		Name:        "echo",
		Description: "Echo the input",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		Message string `json:"message"`
	}) (*mcp.CallToolResult, struct{ Echo string }, error) {
		return nil, struct{ Echo string }{Echo: in.Message}, nil
	})

	// Create paired in-memory transports
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Run server in background
	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx, serverTransport)
	}()

	// Connect client
	client := New("test-client", "v1.0.0", nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	// Test ListTools
	tools, err := session.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "echo" {
		t.Fatalf("expected tool name 'echo', got %q", tools[0].Name)
	}

	// Test CallTool
	result, err := session.CallTool(ctx, "echo", map[string]any{
		"message": "hello world",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check result content
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
}

func TestSessionMCPSession(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "v1.0.0",
	}, nil)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := New("test-client", "v1.0.0", nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	// Verify we can access the underlying session
	mcpSession := session.MCPSession()
	if mcpSession == nil {
		t.Fatal("expected non-nil MCP session")
	}
}

func TestInitializeResult(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "v1.0.0",
	}, nil)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := New("test-client", "v1.0.0", nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer session.Close()

	initResult := session.InitializeResult()
	if initResult == nil {
		t.Fatal("expected non-nil initialize result")
	}

	if initResult.ServerInfo.Name != "test-server" {
		t.Fatalf("expected server name 'test-server', got %q", initResult.ServerInfo.Name)
	}
}
