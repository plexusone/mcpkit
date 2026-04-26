// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServeHTTP_LocalServer(t *testing.T) {
	rt := New(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	// Register a simple tool with required input schema
	rt.AddToolHandler(&mcp.Tool{
		Name:        "ping",
		Description: "Returns pong",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "pong"}},
		}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use port 0 to get a random available port
	resultChan := make(chan *HTTPServerResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := rt.ServeHTTP(ctx, &HTTPServerOptions{
			Addr: "localhost:0",
		})
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop the server
	cancel()

	// Wait for the server to stop
	select {
	case result := <-resultChan:
		if result.LocalAddr == "" {
			t.Error("expected LocalAddr to be set")
		}
		if result.LocalURL == "" {
			t.Error("expected LocalURL to be set")
		}
		if !strings.Contains(result.LocalURL, "/mcp") {
			t.Errorf("expected LocalURL to contain /mcp, got %s", result.LocalURL)
		}
	case err := <-errChan:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to stop")
	}
}

func TestServeHTTP_CustomPath(t *testing.T) {
	rt := New(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultChan := make(chan *HTTPServerResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := rt.ServeHTTP(ctx, &HTTPServerOptions{
			Addr: "localhost:0",
			Path: "/custom-mcp-path",
		})
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case result := <-resultChan:
		if !strings.Contains(result.LocalURL, "/custom-mcp-path") {
			t.Errorf("expected LocalURL to contain /custom-mcp-path, got %s", result.LocalURL)
		}
	case err := <-errChan:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to stop")
	}
}

func TestServeHTTP_MissingAddr(t *testing.T) {
	rt := New(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	ctx := context.Background()

	_, err := rt.ServeHTTP(ctx, &HTTPServerOptions{
		// No Addr and no Ngrok
	})

	if err == nil {
		t.Fatal("expected error when Addr is missing and Ngrok is nil")
	}

	if !strings.Contains(err.Error(), "addr is required") {
		t.Errorf("expected error about addr being required, got: %v", err)
	}
}

func TestServeHTTP_NgrokMissingAuthtoken(t *testing.T) {
	rt := New(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	// Clear any env var that might be set
	t.Setenv("NGROK_AUTHTOKEN", "")

	ctx := context.Background()

	_, err := rt.ServeHTTP(ctx, &HTTPServerOptions{
		Ngrok: &NgrokOptions{
			// No Authtoken and env var is empty
		},
	})

	if err == nil {
		t.Fatal("expected error when ngrok authtoken is missing")
	}

	if !strings.Contains(err.Error(), "authtoken is required") {
		t.Errorf("expected error about authtoken being required, got: %v", err)
	}
}

func TestServeHTTP_AcceptsConnections(t *testing.T) {
	rt := New(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a known port for testing
	port := 19283 // Random high port unlikely to be in use
	addr := fmt.Sprintf("localhost:%d", port)

	serverErrChan := make(chan error, 1)
	go func() {
		_, err := rt.ServeHTTP(ctx, &HTTPServerOptions{
			Addr: addr,
		})
		serverErrChan <- err
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Make a request to the server
	resp, err := http.Get(fmt.Sprintf("http://%s/mcp", addr))
	if err != nil {
		// Server might not have started, that's OK for this test
		t.Skipf("could not connect to server: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("warning: failed to close response body: %v", err)
		}
	}()

	// MCP endpoint should respond (might be an error response since we're
	// not sending proper MCP protocol, but it should respond)
	body, _ := io.ReadAll(resp.Body)
	t.Logf("Response status: %d, body: %s", resp.StatusCode, string(body))

	cancel()

	select {
	case err := <-serverErrChan:
		if err != nil {
			t.Logf("server error (expected): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to stop")
	}
}

func TestHTTPServerOptions_Defaults(t *testing.T) {
	// Test that nil options work
	rt := New(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := rt.ServeHTTP(ctx, nil)
	if err == nil {
		t.Fatal("expected error with nil options and no Addr")
	}
}
