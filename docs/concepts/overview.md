# Concepts Overview

OmniSkill provides a unified infrastructure for AI agent capabilities. This page explains the core concepts.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Application                        │
├─────────────────────────────────────────────────────────────┤
│                        Registry                              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │ Skill A │  │ Skill B │  │ Skill C │  │ Skill D │        │
│  │ (local) │  │ (local) │  │  (MCP)  │  │  (MCP)  │        │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘        │
│       │            │            │            │              │
├───────┴────────────┴────────────┴────────────┴──────────────┤
│                    Skill Interface                           │
│         Name() | Description() | Tools() | Init() | Close() │
└──────────────────────────────────────────────────────────────┘
                              │
            ┌─────────────────┼─────────────────┐
            ▼                 ▼                 ▼
     ┌──────────┐      ┌──────────┐      ┌──────────┐
     │ Library  │      │   MCP    │      │   MCP    │
     │   Mode   │      │  Server  │      │  Client  │
     └──────────┘      └──────────┘      └──────────┘
```

## Core Concepts

### Skills

A **Skill** is a named collection of related tools with lifecycle management. Skills implement the `skill.Skill` interface:

```go
type Skill interface {
    Name() string
    Description() string
    Tools() []Tool
    Init(ctx context.Context) error
    Close() error
}
```

Skills group related functionality. For example, a "weather" skill might have tools for current conditions, forecasts, and alerts.

### Tools

A **Tool** is a single callable function that an AI agent can invoke. Tools have:

- **Name** - Unique identifier
- **Description** - What the tool does (helps AI understand when to use it)
- **Parameters** - Input schema (JSON Schema compatible)
- **Handler** - The function that executes when called

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]Parameter
    Call(ctx context.Context, params map[string]any) (any, error)
}
```

### Registry

The **Registry** is a central place to register and discover skills:

- Register/unregister skills dynamically
- Look up skills by name
- List all available tools across skills
- Manage skill lifecycle (Init/Close)

### Execution Modes

OmniSkill supports multiple execution modes:

| Mode | Description | Use Case |
|------|-------------|----------|
| **Library** | Direct in-process calls | Embedded agents, testing, pipelines |
| **MCP Server** | Expose via MCP protocol | Claude Desktop, external clients |
| **MCP Client** | Consume remote MCP servers | Integrate third-party tools |

## Design Principles

### 1. Define Once, Use Everywhere

Skills and tools are defined using a common interface. The same skill can be:

- Called directly in library mode
- Exposed as an MCP server
- Registered with a central registry

### 2. Protocol at the Edge

MCP is treated as an "edge protocol" for external communication. Internal tool invocation bypasses JSON-RPC overhead for better performance.

### 3. Type Safety

Go generics provide type-safe handlers with automatic JSON schema inference:

```go
runtime.AddTool(rt, tool, func(ctx context.Context, req *mcp.CallToolRequest, in MyInput) (*mcp.CallToolResult, MyOutput, error) {
    // Type-safe input and output
})
```

### 4. Composable

Skills can wrap other skills, MCP sessions, or any source of tools:

```go
// Local skill
localSkill := &skill.BaseSkill{...}

// Remote MCP server as skill
remoteSkill := session.AsSkill()

// Both registered in same registry
reg.Register(localSkill)
reg.Register(remoteSkill)
```

## Next Steps

- [Skills](skills.md) - Creating and managing skills
- [Tools](tools.md) - Defining tools and parameters
- [Packs](packs.md) - Distributing skills as Go modules
- [Registry](registry.md) - Skill registration and discovery
