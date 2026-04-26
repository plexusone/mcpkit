// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package omniskill provides a unified skill/tool infrastructure for AI applications.
//
// OmniSkill is organized into focused subpackages:
//
//   - mcp/server: MCP server runtime with tools, prompts, resources, and
//     multiple transport options (stdio, HTTP, SSE)
//   - mcp/client: MCP client for connecting to external MCP servers
//   - mcp/oauth2: OAuth 2.1 Authorization Server with PKCE support for
//     MCP authentication
//
// # Quick Start
//
// For building MCP servers, import the server package:
//
//	import "github.com/plexusone/omniskill/mcp/server"
//
//	rt := server.New(&mcp.Implementation{
//	    Name:    "my-server",
//	    Version: "1.0.0",
//	}, nil)
//
//	// Add tools, prompts, resources
//	server.AddTool(rt, tool, handler)
//
//	// Library mode: call directly
//	result, err := rt.CallTool(ctx, "add", map[string]any{"a": 1, "b": 2})
//
//	// Server mode: serve via HTTP
//	rt.ServeHTTP(ctx, &server.HTTPServerOptions{
//	    Addr: ":8080",
//	})
//
// For connecting to MCP servers as a client:
//
//	import "github.com/plexusone/omniskill/mcp/client"
//
//	c := client.New("my-app", "1.0.0", nil)
//	session, err := c.ConnectCommand(ctx, exec.Command("npx", "-y", "@modelcontextprotocol/server-github"), nil)
//	tools, err := session.ListTools(ctx)
//
// For OAuth 2.1 authentication with PKCE (required by ChatGPT.com and other
// MCP clients):
//
//	import "github.com/plexusone/omniskill/mcp/oauth2"
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
// OmniSkill enables defining skills once and exposing them across multiple
// formats: MCP protocol, compiled Go skills, OpenAPI, and more.
//
// MCP (Model Context Protocol) is treated as the primary interoperability
// protocol, while providing zero-overhead library-mode for Go consumers.
package omniskill
