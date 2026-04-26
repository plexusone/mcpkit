# Release Notes - v0.5.0

**Release Date:** 2026-04-26

## Overview

This release adds an MCP client package for connecting to external MCP servers. The client wraps the official MCP Go SDK and provides convenient methods for tool discovery and execution.

## Installation

```bash
go get github.com/plexusone/mcpkit@v0.5.0
```

Requires Go 1.26+ and MCP Go SDK v1.5.0+.

## Highlights

- **MCP client package** for connecting to external MCP servers
- **Tool discovery** via `Session.ListTools()`
- **Tool execution** via `Session.CallTool()`
- **Subprocess spawning** via `Client.ConnectCommand()`

## What's New

### Client Package

The new `client/` package enables connecting to MCP servers as a client:

```go
import "github.com/plexusone/mcpkit/client"

// Create a client
c := client.New("my-app", "1.0.0", nil)

// Connect to an MCP server via subprocess
cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-github")
session, err := c.ConnectCommand(ctx, cmd, nil)
if err != nil {
    return err
}
defer session.Close()

// Discover available tools
tools, err := session.ListTools(ctx)
for _, tool := range tools {
    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
}

// Call a tool
result, err := session.CallTool(ctx, "search_repositories", map[string]any{
    "query": "language:go stars:>1000",
})
```

### New Types

| Type | Description |
|------|-------------|
| `client.Client` | MCP client with connection methods |
| `client.Session` | Active session with tool operations |
| `client.Options` | Client configuration options |

### New Methods

| Method | Description |
|--------|-------------|
| `client.New()` | Create a new MCP client |
| `Client.Connect()` | Connect with any MCP transport |
| `Client.ConnectCommand()` | Spawn subprocess and connect via stdio |
| `Session.ListTools()` | Discover available tools |
| `Session.CallTool()` | Execute a tool by name |
| `Session.MCPSession()` | Access underlying MCP SDK session |
| `Session.Close()` | Close the session |

## Upgrade Guide

### From v0.4.x

No breaking changes. Simply update your dependency:

```bash
go get github.com/plexusone/mcpkit@v0.5.0
go mod tidy
```

## Use Cases

### Connecting to MCP Servers

```go
// GitHub MCP server
cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-github")
cmd.Env = append(os.Environ(), "GITHUB_TOKEN="+token)
session, _ := client.ConnectCommand(ctx, cmd, nil)

// Filesystem MCP server
cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir")
session, _ := client.ConnectCommand(ctx, cmd, nil)
```

### Custom Transport

```go
// Connect with custom transport (e.g., SSE)
transport := mcp.NewSSEClientTransport(serverURL)
session, err := c.Connect(ctx, transport, nil)
```

## Contributors

- John Wang
- Claude Opus 4.5

## Links

- [GitHub Repository](https://github.com/plexusone/mcpkit)
- [Go Package Documentation](https://pkg.go.dev/github.com/plexusone/mcpkit)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
