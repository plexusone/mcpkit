# OmniSkill

**Unified skill infrastructure for AI agents in Go.**

OmniSkill provides a common interface for defining, registering, and invoking AI agent capabilities ("skills") across multiple execution environments:

- **MCP Servers** - Expose skills via Model Context Protocol
- **MCP Clients** - Consume remote MCP servers as skills
- **Library Mode** - Direct in-process invocation without protocol overhead
- **Registry** - Central skill discovery and lifecycle management

## Key Features

- **Unified Skill Interface** - Define tools once, use everywhere
- **MCP Integration** - Full MCP protocol support for both server and client
- **Zero-Overhead Library Mode** - Call tools directly without JSON-RPC
- **OAuth 2.1 Support** - Built-in authentication for public MCP servers
- **Type-Safe Handlers** - Generic handlers with automatic schema inference

## Quick Example

```go
package main

import (
    "context"
    "fmt"

    "github.com/plexusone/omniskill/mcp/server"
    "github.com/plexusone/omniskill/skill"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    // Create a skill with tools
    mathSkill := &skill.BaseSkill{
        SkillName:        "math",
        SkillDescription: "Mathematical operations",
        SkillTools: []skill.Tool{
            skill.NewTool("add", "Add two numbers",
                map[string]skill.Parameter{
                    "a": {Type: "number", Required: true},
                    "b": {Type: "number", Required: true},
                },
                func(ctx context.Context, params map[string]any) (any, error) {
                    a := params["a"].(float64)
                    b := params["b"].(float64)
                    return map[string]any{"sum": a + b}, nil
                },
            ),
        },
    }

    // Create runtime and register skill
    rt := server.New(&mcp.Implementation{
        Name:    "calculator",
        Version: "1.0.0",
    }, nil)

    rt.RegisterSkill(mathSkill)

    // Library mode - call directly
    result, _ := rt.CallTool(context.Background(), "add", map[string]any{"a": 1.0, "b": 2.0})
    fmt.Println(result) // {"sum": 3}

    // Or serve via MCP
    // rt.ServeStdio(context.Background())
}
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

## Getting Started

- [Installation](getting-started/installation.md)
- [Quick Start](getting-started/quickstart.md)

## Learn More

- [Concepts Overview](concepts/overview.md)
- [Skills](concepts/skills.md)
- [Tools](concepts/tools.md)
- [Registry](concepts/registry.md)
