// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package mcpkit provides a toolkit for building MCP (Model Context Protocol)
// applications in Go.
//
// MCPKit is organized into focused subpackages:
//
//   - runtime: Core MCP server runtime with tools, prompts, resources, and
//     multiple transport options (stdio, HTTP, SSE)
//   - oauth2: OAuth 2.1 Authorization Server with PKCE support for
//     MCP authentication
//
// # Quick Start
//
// For building MCP servers, import the runtime package:
//
//	import "github.com/plexusone/mcpkit/runtime"
//
//	rt := runtime.New(&mcp.Implementation{
//	    Name:    "my-server",
//	    Version: "1.0.0",
//	}, nil)
//
//	// Add tools, prompts, resources
//	runtime.AddTool(rt, tool, handler)
//
//	// Library mode: call directly
//	result, err := rt.CallTool(ctx, "add", map[string]any{"a": 1, "b": 2})
//
//	// Server mode: serve via HTTP
//	rt.ServeHTTP(ctx, &runtime.HTTPServerOptions{
//	    Addr: ":8080",
//	})
//
// For OAuth 2.1 authentication with PKCE (required by ChatGPT.com and other
// MCP clients):
//
//	import "github.com/plexusone/mcpkit/oauth2"
//
//	srv, err := oauth2.New(&oauth2.Config{
//	    Issuer: "https://example.com",
//	    Users:  map[string]string{"admin": "password"},
//	})
//
// See the individual package documentation for detailed usage.
//
// # Design Philosophy
//
// MCP (Model Context Protocol) is fundamentally a client-server protocol based
// on JSON-RPC. However, many use cases benefit from invoking MCP capabilities
// directly in-process without the overhead of transport serialization:
//
//   - Unit testing tools without mocking transports
//   - Embedding agent capabilities in applications
//   - Building local pipelines
//   - Serverless runtimes
//
// mcpkit treats MCP as an "edge protocol" while providing a library-first
// internal API. Tools registered with mcpkit use the exact same handler
// signatures as the MCP SDK, ensuring behavior is identical regardless of
// execution mode.
package mcpkit
