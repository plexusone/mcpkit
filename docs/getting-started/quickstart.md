# Quick Start

This guide walks you through creating a skill and using it in different modes.

## Create a Skill

Skills are collections of related tools. Here's a simple calculator skill:

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
            "a": {Type: "number", Description: "First number", Required: true},
            "b": {Type: "number", Description: "Second number", Required: true},
        },
        func(ctx context.Context, params map[string]any) (any, error) {
            a := params["a"].(float64)
            b := params["b"].(float64)
            return map[string]any{"sum": a + b}, nil
        },
    )

    // Create a skill with the tool
    mathSkill := &skill.BaseSkill{
        SkillName:        "math",
        SkillDescription: "Mathematical operations",
        SkillTools:       []skill.Tool{addTool},
    }

    // Use the skill...
}
```

## Library Mode

Call tools directly without any protocol overhead:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/plexusone/omniskill/mcp/server"
    "github.com/plexusone/omniskill/skill"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    // Create runtime
    rt := server.New(&mcp.Implementation{
        Name:    "calculator",
        Version: "1.0.0",
    }, nil)

    // Create and register skill
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

    rt.RegisterSkill(&skill.BaseSkill{
        SkillName:  "math",
        SkillTools: []skill.Tool{addTool},
    })

    // Call tool directly
    result, err := rt.CallTool(context.Background(), "add", map[string]any{
        "a": 5.0,
        "b": 3.0,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %v\n", result.Content[0])
}
```

## MCP Server Mode

Expose skills via MCP for Claude Desktop or other MCP clients:

```go
package main

import (
    "context"
    "log"

    "github.com/plexusone/omniskill/mcp/server"
    "github.com/plexusone/omniskill/skill"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    rt := server.New(&mcp.Implementation{
        Name:    "calculator",
        Version: "1.0.0",
    }, nil)

    // Register skills...
    rt.RegisterSkill(createMathSkill())

    // Serve over stdio (for Claude Desktop)
    if err := rt.ServeStdio(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

Add to Claude Desktop config (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "calculator": {
      "command": "/path/to/your/binary"
    }
  }
}
```

## MCP Client Mode

Connect to remote MCP servers and use them as skills:

```go
package main

import (
    "context"
    "log"
    "os/exec"

    "github.com/plexusone/omniskill/mcp/client"
)

func main() {
    ctx := context.Background()

    // Create client
    c := client.New("my-app", "1.0.0", nil)

    // Connect to an MCP server
    cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp")
    session, err := c.ConnectCommand(ctx, cmd)
    if err != nil {
        log.Fatal(err)
    }
    defer session.Close()

    // Wrap session as a skill
    fsSkill := session.AsSkill(
        client.WithSkillName("filesystem"),
        client.WithSkillDescription("File system operations"),
    )

    // Use tools from the remote server
    for _, tool := range fsSkill.Tools() {
        log.Printf("Tool: %s - %s", tool.Name(), tool.Description())
    }
}
```

## Using the Registry

Register and discover skills centrally:

```go
package main

import (
    "context"
    "log"

    "github.com/plexusone/omniskill/registry"
    "github.com/plexusone/omniskill/skill"
)

func main() {
    // Create registry
    reg := registry.New()

    // Register skills
    reg.Register(&skill.BaseSkill{SkillName: "math"})
    reg.Register(&skill.BaseSkill{SkillName: "weather"})

    // List all skills
    for _, s := range reg.List() {
        log.Printf("Skill: %s", s.Name())
    }

    // Get specific skill
    mathSkill, err := reg.Get("math")
    if err != nil {
        log.Fatal(err)
    }

    // Initialize all skills
    if err := reg.Init(context.Background()); err != nil {
        log.Fatal(err)
    }

    // Cleanup
    defer reg.Close()
}
```

## Next Steps

- [Concepts Overview](../concepts/overview.md) - Understand the architecture
- [Skills](../concepts/skills.md) - Deep dive into skills
- [MCP Server](../mcp/server.md) - Server configuration options
