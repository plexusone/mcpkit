# OmniSkill

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omniskill/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omniskill/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omniskill/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omniskill/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omniskill/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omniskill/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omniskill
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omniskill
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omniskill
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omniskill
 [docs-mkdoc-svg]: https://img.shields.io/badge/docs-guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.dev/omniskill
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omniskill/blob/main/LICENSE

Unified skill infrastructure for AI agents in Go.

## Overview

OmniSkill provides a common interface for defining, registering, and invoking AI agent capabilities across multiple execution environments:

- **skill/** - Core Skill and Tool interfaces
- **registry/** - Skill registration and discovery
- **mcp/server/** - MCP server runtime with tools, prompts, resources
- **mcp/client/** - MCP client for connecting to remote servers
- **mcp/oauth2/** - OAuth 2.1 Authorization Server for authenticated MCP

Skills can be invoked via:

- **Library mode** - Direct in-process calls without protocol overhead
- **MCP Server** - Expose via Model Context Protocol (stdio, HTTP, SSE)
- **MCP Client** - Consume remote MCP servers as local skills

## Installation

```bash
go get github.com/plexusone/omniskill
```

## Quick Start

### Define a Skill

```go
package main

import (
    "context"
    "github.com/plexusone/omniskill/skill"
)

func main() {
    // Create a tool
    addTool := skill.NewTool("add", "Add two numbers",
        map[string]skill.Parameter{
            "a": {Type: "number", Required: true},
            "b": {Type: "number", Required: true},
        },
        func(ctx context.Context, params map[string]any) (any, error) {
            a := params["a"].(float64)
            b := params["b"].(float64)
            return map[string]any{"sum": a + b}, nil
        },
    )

    // Create a skill
    mathSkill := &skill.BaseSkill{
        SkillName:        "math",
        SkillDescription: "Mathematical operations",
        SkillTools:       []skill.Tool{addTool},
    }
}
```

### Library Mode

Call tools directly without MCP overhead:

```go
import (
    "github.com/plexusone/omniskill/mcp/server"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

rt := server.New(&mcp.Implementation{
    Name:    "calculator",
    Version: "1.0.0",
}, nil)

rt.RegisterSkill(mathSkill)

// Direct invocation - no JSON-RPC, no transport
result, err := rt.CallTool(ctx, "add", map[string]any{"a": 1.0, "b": 2.0})
```

### MCP Server Mode

Expose skills via MCP for Claude Desktop or other clients:

```go
// stdio (for Claude Desktop)
rt.ServeStdio(ctx)

// HTTP with SSE
rt.ServeHTTP(ctx, &server.HTTPServerOptions{Addr: ":8080"})

// With OAuth2 authentication (for ChatGPT.com)
rt.ServeHTTP(ctx, &server.HTTPServerOptions{
    Addr: ":8080",
    OAuth2: &server.OAuth2Options{
        Users: map[string]string{"admin": "password"},
    },
})
```

### MCP Client Mode

Connect to remote MCP servers and use them as skills:

```go
import (
    "os/exec"
    "github.com/plexusone/omniskill/mcp/client"
)

c := client.New("my-app", "1.0.0", nil)

// Connect to MCP server
cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp")
session, err := c.ConnectCommand(ctx, cmd)
defer session.Close()

// Wrap as skill
fsSkill := session.AsSkill(client.WithSkillName("filesystem"))

// Use like any local skill
for _, tool := range fsSkill.Tools() {
    fmt.Println(tool.Name())
}
```

### Registry

Central skill registration and discovery:

```go
import "github.com/plexusone/omniskill/registry"

reg := registry.New()
reg.Register(mathSkill)
reg.Register(fsSkill)

// Discover all tools
for _, tool := range reg.ListTools() {
    fmt.Printf("%s: %s\n", tool.Name(), tool.Description())
}

// Initialize all skills
reg.Init(ctx)
defer reg.Close()
```

### Auto-Registration

Skills registered with the runtime can auto-register with a registry:

```go
reg := registry.New()
rt := server.New(impl, &server.Options{
    Registry: reg,  // Enable auto-registration
})

rt.RegisterSkill(mathSkill)  // Also registers with reg
```

## Package Structure

```
github.com/plexusone/omniskill
├── skill/       # Core Skill and Tool interfaces
├── registry/    # Skill registration and discovery
├── mcp/
│   ├── server/  # MCP server runtime
│   ├── client/  # MCP client for remote servers
│   └── oauth2/  # OAuth 2.1 authorization server
└── doc.go
```

## Documentation

- [Getting Started](https://plexusone.dev/omniskill/getting-started/installation/)
- [Concepts](https://plexusone.dev/omniskill/concepts/overview/)
- [MCP Server](https://plexusone.dev/omniskill/mcp/server/)
- [MCP Client](https://plexusone.dev/omniskill/mcp/client/)
- [API Reference](https://pkg.go.dev/github.com/plexusone/omniskill)

## Feature Comparison

| Feature | Library Mode | MCP Server | MCP Client |
|---------|:------------:|:----------:|:----------:|
| Direct tool calls | ✓ | - | - |
| JSON-RPC overhead | None | Yes | Yes |
| Claude Desktop | - | ✓ | - |
| Remote servers | - | - | ✓ |
| Skill interface | ✓ | ✓ | ✓ |

## Design Philosophy

1. **Define Once, Use Everywhere** - Skills work in library mode, as MCP servers, or wrapping MCP clients
2. **Protocol at the Edge** - MCP is for external communication; internal calls bypass JSON-RPC
3. **Type Safety** - Generic handlers with automatic JSON schema inference
4. **Composable** - Skills can wrap other skills or remote MCP sessions

## License

MIT License - see LICENSE file for details.
