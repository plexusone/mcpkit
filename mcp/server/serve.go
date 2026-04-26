// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServeStdio runs the runtime as an MCP server over stdio transport.
//
// This is the standard way to run an MCP server as a subprocess. The server
// communicates with the client via stdin/stdout using newline-delimited JSON.
//
// ServeStdio blocks until the client terminates the connection or the context
// is cancelled.
//
// Example:
//
//	func main() {
//		rt := mcpkit.New(&mcp.Implementation{Name: "my-server", Version: "v1.0.0"}, nil)
//		// ... register tools ...
//		if err := rt.ServeStdio(context.Background()); err != nil {
//			log.Fatal(err)
//		}
//	}
func (r *Runtime) ServeStdio(ctx context.Context) error {
	return r.server.Run(ctx, &mcp.StdioTransport{})
}

// ServeIO runs the runtime as an MCP server over custom IO streams.
//
// This is useful for testing or when you need to control the IO streams
// directly rather than using stdin/stdout.
func (r *Runtime) ServeIO(ctx context.Context, reader io.ReadCloser, writer io.WriteCloser) error {
	transport := &mcp.IOTransport{
		Reader: reader,
		Writer: writer,
	}
	return r.server.Run(ctx, transport)
}

// Serve runs the runtime with a custom MCP transport.
//
// This is the most flexible option, allowing any transport that implements
// the mcp.Transport interface.
func (r *Runtime) Serve(ctx context.Context, transport mcp.Transport) error {
	return r.server.Run(ctx, transport)
}

// Connect creates a session for a single connection.
//
// Unlike [Runtime.ServeStdio] which runs a blocking loop, Connect returns
// immediately with a session that can be used to await client termination
// or manage the connection lifecycle.
//
// This is useful for HTTP-based transports or when managing multiple
// concurrent sessions.
func (r *Runtime) Connect(ctx context.Context, transport mcp.Transport) (*mcp.ServerSession, error) {
	return r.server.Connect(ctx, transport, nil)
}

// StreamableHTTPHandler returns an http.Handler for MCP's Streamable HTTP transport.
//
// This enables serving MCP over HTTP using Server-Sent Events (SSE) for
// server-to-client messages. The handler can be mounted on any HTTP server.
//
// Example:
//
//	rt := mcpkit.New(&mcp.Implementation{Name: "my-server", Version: "v1.0.0"}, nil)
//	// ... register tools ...
//	http.Handle("/mcp", rt.StreamableHTTPHandler(nil))
//	http.ListenAndServe(":8080", nil)
func (r *Runtime) StreamableHTTPHandler(opts *mcp.StreamableHTTPOptions) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return r.server
	}, opts)
}

// SSEHandler returns an http.Handler for the legacy SSE transport.
//
// This is provided for backwards compatibility with older MCP clients.
// New implementations should prefer [Runtime.StreamableHTTPHandler].
func (r *Runtime) SSEHandler(opts *mcp.SSEOptions) http.Handler {
	return mcp.NewSSEHandler(func(*http.Request) *mcp.Server {
		return r.server
	}, opts)
}

// InMemorySession creates an in-memory client-server session pair.
//
// This is useful for testing or for scenarios where you want MCP semantics
// (including JSON-RPC serialization) but don't need network transport.
//
// Returns the server session and client session. The caller should close
// the client session when done, which will also terminate the server session.
//
// Example:
//
//	serverSession, clientSession, err := rt.InMemorySession(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer clientSession.Close()
//
//	// Use clientSession to call tools via MCP protocol
//	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "add", Arguments: map[string]any{"a": 1, "b": 2}})
func (r *Runtime) InMemorySession(ctx context.Context) (*mcp.ServerSession, *mcp.ClientSession, error) {
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := r.server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, nil, err
	}

	client := mcp.NewClient(r.impl, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close() // Best-effort cleanup; already returning an error
		return nil, nil, err
	}

	return serverSession, clientSession, nil
}
