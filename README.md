# MCPKit

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

A toolkit for building MCP (Model Context Protocol) applications in Go.

## Overview

MCPKit wraps the official [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) to provide focused packages for building MCP servers:

- **runtime**: Core MCP server runtime with tools, prompts, resources, and multiple transport options (stdio, HTTP, SSE)
- **oauth2**: OAuth 2.1 Authorization Server with PKCE support for MCP authentication (required by ChatGPT.com)

The runtime package provides a unified API where tools, prompts, and resources are defined once and can be invoked either:

- **Library mode**: Direct in-process function calls without JSON-RPC overhead
- **Server mode**: Standard MCP transports (stdio, HTTP, SSE)

## Installation

```bash
go get github.com/plexusone/mcpkit
```

## Quick Start

### Library Mode Example

Use tools directly in your application without MCP transport overhead:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/plexusone/mcpkit/runtime"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type AddInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

type AddOutput struct {
    Sum int `json:"sum"`
}

func main() {
    rt := runtime.New(&mcp.Implementation{
        Name:    "calculator",
        Version: "v1.0.0",
    }, nil)

    runtime.AddTool(rt, &mcp.Tool{
        Name:        "add",
        Description: "Add two numbers",
    }, func(ctx context.Context, req *mcp.CallToolRequest, in AddInput) (*mcp.CallToolResult, AddOutput, error) {
        return nil, AddOutput{Sum: in.A + in.B}, nil
    })

    // Call tool directly - no JSON-RPC, no transport
    result, err := rt.CallTool(context.Background(), "add", map[string]any{"a": 1, "b": 2})
    if err != nil {
        log.Fatal(err)
    }

    text := result.Content[0].(*mcp.TextContent).Text
    fmt.Println(text) // Output: {"sum":3}
}
```

### Server Mode Example (stdio)

Expose the same tools as an MCP server for Claude Desktop or other MCP clients:

```go
package main

import (
    "context"
    "log"

    "github.com/plexusone/mcpkit/runtime"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type AddInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

type AddOutput struct {
    Sum int `json:"sum"`
}

func main() {
    rt := runtime.New(&mcp.Implementation{
        Name:    "calculator",
        Version: "v1.0.0",
    }, nil)

    runtime.AddTool(rt, &mcp.Tool{
        Name:        "add",
        Description: "Add two numbers",
    }, func(ctx context.Context, req *mcp.CallToolRequest, in AddInput) (*mcp.CallToolResult, AddOutput, error) {
        return nil, AddOutput{Sum: in.A + in.B}, nil
    })

    // Run as MCP server over stdio
    if err := rt.ServeStdio(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

### Server Mode Example (HTTP)

Expose tools over HTTP with SSE for server-to-client messages:

```go
package main

import (
    "log"
    "net/http"

    "github.com/plexusone/mcpkit/runtime"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    rt := runtime.New(&mcp.Implementation{
        Name:    "calculator",
        Version: "v1.0.0",
    }, nil)

    // Register tools...

    http.Handle("/mcp", rt.StreamableHTTPHandler(nil))
    log.Println("MCP server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### HTTP Server with OAuth 2.1 Authentication

For public MCP servers that need authentication (required by ChatGPT.com):

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/plexusone/mcpkit/runtime"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    rt := runtime.New(&mcp.Implementation{
        Name:    "my-server",
        Version: "v1.0.0",
    }, nil)

    // Register tools...

    ctx := context.Background()
    result, err := rt.ServeHTTP(ctx, &runtime.HTTPServerOptions{
        Addr: ":8080",
        OAuth2: &runtime.OAuth2Options{
            Users: map[string]string{"admin": "password"},
        },
        OnReady: func(r *runtime.HTTPServerResult) {
            fmt.Printf("MCP endpoint: %s\n", r.LocalURL)
            fmt.Printf("OAuth2 Client ID: %s\n", r.OAuth2.ClientID)
            fmt.Printf("OAuth2 Client Secret: %s\n", r.OAuth2.ClientSecret)
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    _ = result
}
```

## Design Philosophy

MCP (Model Context Protocol) is fundamentally a client-server protocol based on JSON-RPC. However, many use cases benefit from invoking MCP capabilities directly in-process:

- Unit testing without mocking transports
- Embedding agent capabilities in applications
- Building local pipelines
- Serverless runtimes

MCPKit treats MCP as an "edge protocol" while providing a library-first internal API. Tools registered with MCPKit use the exact same handler signatures as the MCP SDK, ensuring behavior is identical regardless of execution mode.

## Key Features

### Same Handlers, Two Modes

Tools, prompts, and resources are defined once using MCP SDK types:

```go
// Register tool
runtime.AddTool(rt, &mcp.Tool{Name: "calculate"}, handler)

// Library mode
result, err := rt.CallTool(ctx, "calculate", args)

// Server mode
rt.ServeStdio(ctx)
```

### Full MCP SDK Compatibility

- Uses `mcp.Tool`, `mcp.Prompt`, `mcp.Resource` types directly
- Typed handlers with automatic schema inference via `AddTool[In, Out]`
- All MCP transports supported (stdio, HTTP, SSE)

### Transport Adapters

```go
// Stdio (subprocess)
rt.ServeStdio(ctx)

// HTTP/SSE
http.Handle("/mcp", rt.StreamableHTTPHandler(nil))

// In-memory (testing)
_, clientSession, _ := rt.InMemorySession(ctx)
```

## Feature Comparison: Library vs Server Mode

| Feature | Library Mode | Server Mode | Notes |
|---------|:------------:|:-----------:|-------|
| Tools | Yes | Yes | Full parity |
| Prompts | Yes | Yes | Full parity |
| Static Resources | Yes | Yes | Full parity |
| Resource Templates | No | Yes | See below |
| JSON-RPC overhead | None | Yes | Library mode is faster |
| MCP client required | No | Yes | Library mode is standalone |

### Static vs Dynamic Resource Templates

**Static resources** have fixed URIs and work identically in both modes:

```go
rt.AddResource(&mcp.Resource{
    URI:  "config://app/settings",
    Name: "settings",
}, handler)

// Library mode
rt.ReadResource(ctx, "config://app/settings")

// Server mode - same handler, MCP protocol
```

**Dynamic resource templates** use RFC 6570 URI Template syntax for pattern matching:

```go
rt.AddResourceTemplate(&mcp.ResourceTemplate{
    URITemplate: "file:///{+path}",  // {+path} can contain /
}, handler)

// Matches: file:///docs/readme.md, file:///src/main.go, etc.
```

Resource templates are registered with the MCP server and work in server mode. Library-mode dispatch (`ReadResource`) currently supports exact URI matches only. For template matching in library mode, use `MCPServer()` directly.

**Important:** The URI scheme (e.g., `file:///`) is just an identifier—it doesn't mean the resource is on the filesystem. Your handler determines what content is returned:

```go
// This "file:///" resource returns computed content, not filesystem data
rt.AddResourceTemplate(&mcp.ResourceTemplate{
    URITemplate: "file:///{+path}",
}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    // You decide: read from disk, database, return hardcoded data, etc.
    return &mcp.ReadResourceResult{
        Contents: []*mcp.ResourceContents{{
            URI:  req.Params.URI,
            Text: "This content is computed, not from a file",
        }},
    }, nil
})
```

## MCP Feature Adoption

Based on MCP ecosystem patterns, feature adoption varies significantly:

| Feature | Adoption | Recommendation |
|---------|----------|----------------|
| Tools | ~80% of servers | Primary focus |
| Static Resources | ~50% | Use when needed |
| Prompts | ~10-20% | Optional |
| Resource Templates | Rare | Usually unnecessary |

**Why tools dominate:** Tools perform actions and return results—they cover most use cases. Resources are better suited for data that clients may cache or subscribe to.

**Why templates are uncommon:**

1. **Tools with parameters are simpler** - Instead of `file:///{+path}`, use a `read_file` tool with a `path` parameter
2. **Static resources usually suffice** - A few fixed URIs cover most configuration/data needs
3. **Added complexity** - Template matching and URI parsing add overhead for little benefit

**When to use resource templates:**

- File browsers where URI semantics matter to the client
- REST-like resource hierarchies
- When clients need resource-specific features (subscriptions, caching hints)

## Package Structure

```
github.com/plexusone/mcpkit
├── runtime/     # Core MCP server runtime
│   ├── Runtime type (New, CallTool, ServeStdio, ServeHTTP, etc.)
│   ├── Tool, Prompt, Resource registration
│   └── OAuth options for HTTP serving
├── oauth2/      # OAuth 2.1 Authorization Server
│   ├── Authorization Code Flow with PKCE (RFC 7636)
│   ├── Dynamic Client Registration (RFC 7591)
│   └── Authorization Server Metadata (RFC 8414)
└── doc.go       # Package documentation
```

## API Reference

### Runtime Creation

```go
rt := runtime.New(impl *mcp.Implementation, opts *runtime.Options)
```

### Tool Registration

```go
// Generic (with schema inference)
runtime.AddTool(rt, tool *mcp.Tool, handler ToolHandlerFor[In, Out])

// Low-level
rt.AddToolHandler(tool *mcp.Tool, handler mcp.ToolHandler)
```

### Library Mode Invocation

```go
result, err := rt.CallTool(ctx, name string, args any)
result, err := rt.GetPrompt(ctx, name string, args map[string]string)
result, err := rt.ReadResource(ctx, uri string)
```

### Server Mode

```go
rt.ServeStdio(ctx)
rt.ServeIO(ctx, reader, writer)
rt.Serve(ctx, transport)
rt.ServeHTTP(ctx, opts *runtime.HTTPServerOptions) // blocks until context cancelled
rt.StreamableHTTPHandler(opts) // returns http.Handler
rt.SSEHandler(opts)            // returns http.Handler
```

### Inspection

```go
rt.ListTools() []*mcp.Tool
rt.ListPrompts() []*mcp.Prompt
rt.ListResources() []*mcp.Resource
rt.HasTool(name) bool
rt.ToolCount() int
```

## License

MIT License - see LICENSE file for details.

 [go-ci-svg]: https://github.com/plexusone/mcpkit/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/mcpkit/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/mcpkit/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/mcpkit/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/mcpkit/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/mcpkit/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/mcpkit
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/mcpkit
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/mcpkit
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/mcpkit
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fmcpkit
 [loc-svg]: https://tokei.rs/b1/github/plexusone/mcpkit
 [repo-url]: https://github.com/plexusone/mcpkit
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/mcpkit/blob/master/LICENSE
