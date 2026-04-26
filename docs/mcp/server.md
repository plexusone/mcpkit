# MCP Server

The `mcp/server` package provides the MCP server runtime. It wraps the official MCP Go SDK with focused APIs for building MCP servers.

## Overview

The Runtime type is the core of the MCP server. It supports:

- **Library mode** - Direct in-process tool invocation
- **Server mode** - MCP protocol over stdio, HTTP, or SSE
- **Skill registration** - Register skills and their tools
- **OAuth 2.1** - Built-in authentication for public servers

## Creating a Runtime

```go
import (
    "github.com/plexusone/omniskill/mcp/server"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

rt := server.New(&mcp.Implementation{
    Name:    "my-server",
    Version: "1.0.0",
}, nil)
```

## Options

```go
rt := server.New(impl, &server.Options{
    // Custom logger
    Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),

    // MCP SDK server options
    ServerOptions: &mcp.ServerOptions{
        // ...
    },

    // Auto-register skills with this registry
    Registry: registry.New(),
})
```

## Registering Skills

```go
// Register skill (tools use their own names)
rt.RegisterSkill(mathSkill)

// Register with prefix (tools become "skillname_toolname")
rt.RegisterSkillWithPrefix(weatherSkill)
```

## Registering Tools Directly

For MCP-style tool registration with typed handlers:

```go
type AddInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

type AddOutput struct {
    Sum int `json:"sum"`
}

server.AddTool(rt, &mcp.Tool{
    Name:        "add",
    Description: "Add two numbers",
}, func(ctx context.Context, req *mcp.CallToolRequest, in AddInput) (*mcp.CallToolResult, AddOutput, error) {
    return nil, AddOutput{Sum: in.A + in.B}, nil
})
```

## Library Mode

Call tools directly without protocol overhead:

```go
// Call tool
result, err := rt.CallTool(ctx, "add", map[string]any{"a": 1, "b": 2})

// Get prompt
result, err := rt.GetPrompt(ctx, "greeting", map[string]string{"name": "Alice"})

// Read resource
result, err := rt.ReadResource(ctx, "config://app/settings")
```

## Server Mode: Stdio

For Claude Desktop and subprocess-based clients:

```go
if err := rt.ServeStdio(ctx); err != nil {
    log.Fatal(err)
}
```

Claude Desktop configuration:

```json
{
  "mcpServers": {
    "my-server": {
      "command": "/path/to/binary"
    }
  }
}
```

## Server Mode: HTTP

### Basic HTTP Server

```go
result, err := rt.ServeHTTP(ctx, &server.HTTPServerOptions{
    Addr: ":8080",
    Path: "/mcp",  // Optional, default is "/"
})
```

### HTTP Handler for Custom Server

```go
http.Handle("/mcp", rt.StreamableHTTPHandler(nil))
log.Fatal(http.ListenAndServe(":8080", nil))
```

### With ngrok Tunnel

```go
result, err := rt.ServeHTTP(ctx, &server.HTTPServerOptions{
    Addr:          ":8080",
    NgrokAuthtoken: os.Getenv("NGROK_AUTHTOKEN"),
    OnReady: func(r *server.HTTPServerResult) {
        fmt.Printf("Public URL: %s\n", r.PublicURL)
    },
})
```

## Server Mode: SSE

Server-Sent Events for real-time communication:

```go
http.Handle("/mcp", rt.SSEHandler(nil))
```

## OAuth 2.1 Authentication

For public servers that need authentication (required by ChatGPT.com):

```go
result, err := rt.ServeHTTP(ctx, &server.HTTPServerOptions{
    Addr: ":8080",
    OAuth2: &server.OAuth2Options{
        // Simple username/password authentication
        Users: map[string]string{
            "admin": "password",
        },
    },
    OnReady: func(r *server.HTTPServerResult) {
        fmt.Printf("Client ID: %s\n", r.OAuth2.ClientID)
        fmt.Printf("Client Secret: %s\n", r.OAuth2.ClientSecret)
    },
})
```

OAuth2 endpoints are automatically configured:

- `/.well-known/oauth-authorization-server` - Server metadata
- `/authorize` - Authorization endpoint
- `/token` - Token endpoint
- `/register` - Dynamic client registration

## Inspection

```go
// List registered tools
tools := rt.ListTools()

// Check if tool exists
if rt.HasTool("add") {
    // ...
}

// Count tools
count := rt.ToolCount()
```

## Prompts and Resources

### Prompts

```go
rt.AddPrompt(&mcp.Prompt{
    Name:        "greeting",
    Description: "Generate a greeting",
}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    name := req.Params.Arguments["name"]
    return &mcp.GetPromptResult{
        Messages: []*mcp.PromptMessage{{
            Role:    "user",
            Content: &mcp.TextContent{Text: "Hello, " + name},
        }},
    }, nil
})
```

### Resources

```go
rt.AddResource(&mcp.Resource{
    URI:  "config://app/settings",
    Name: "settings",
}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    return &mcp.ReadResourceResult{
        Contents: []*mcp.ResourceContents{{
            URI:  req.Params.URI,
            Text: `{"theme": "dark", "language": "en"}`,
        }},
    }, nil
})
```

## Advanced: Access Underlying Server

```go
mcpServer := rt.MCPServer()
// Use mcp.Server methods directly
```
