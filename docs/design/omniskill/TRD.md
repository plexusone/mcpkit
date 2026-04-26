# Technical Requirements Document: OmniSkill

## Architecture Overview

```
omniskill/
├── skill/              # Core skill/tool interfaces
│   ├── skill.go        # Skill interface
│   ├── tool.go         # Tool interface
│   └── registry.go     # Skill registry
├── mcp/
│   ├── server/         # MCP server (from mcpkit/runtime)
│   │   └── runtime.go
│   ├── client/         # MCP client (from mcpkit/client)
│   │   ├── client.go
│   │   └── session.go
│   └── oauth2/         # OAuth2 server (from mcpkit/oauth2)
├── openapi/            # OpenAPI → skill generation
│   ├── parser.go
│   └── generator.go
├── export/             # Export to various formats
│   ├── compiled/       # Go compiled.Skill format
│   ├── openclaw/       # OpenClaw format
│   └── claudecode/     # Claude Code format
└── registry/           # Skill registry
    ├── registry.go
    └── discovery.go
```

## Core Interfaces

### Skill Interface

```go
package skill

// Skill represents a named collection of related tools.
type Skill interface {
    // Name returns the skill identifier (e.g., "github", "weather").
    Name() string

    // Description returns a human-readable description.
    Description() string

    // Tools returns the tools provided by this skill.
    Tools() []Tool

    // Init initializes the skill (called once before use).
    Init(ctx context.Context) error

    // Close releases any resources held by the skill.
    Close() error
}
```

### Tool Interface

```go
package skill

// Tool represents a single callable function.
type Tool interface {
    // Name returns the tool identifier.
    Name() string

    // Description returns a human-readable description.
    Description() string

    // Parameters returns the JSON Schema for input parameters.
    Parameters() map[string]Parameter

    // Call executes the tool with the given parameters.
    Call(ctx context.Context, params map[string]any) (any, error)
}

// Parameter describes a tool parameter.
type Parameter struct {
    Type        string   // "string", "number", "boolean", "object", "array"
    Description string
    Required    bool
    Enum        []any
    Default     any
}
```

### Registry Interface

```go
package registry

// Registry manages skill discovery and registration.
type Registry interface {
    // Register adds a skill to the registry.
    Register(skill skill.Skill) error

    // Unregister removes a skill from the registry.
    Unregister(name string) error

    // Get returns a skill by name.
    Get(name string) (skill.Skill, error)

    // List returns all registered skills.
    List() []skill.Skill

    // ListTools returns all tools across all skills.
    ListTools() []skill.Tool
}
```

## MCP Integration

### Server (from mcpkit/runtime)

The existing `runtime/` package becomes `mcp/server/`:

```go
package server

// Runtime wraps mcp.Server with skill integration.
type Runtime struct {
    server   *mcp.Server
    skills   []skill.Skill
    registry registry.Registry
}

// New creates a new MCP server runtime.
func New(name, version string, opts ...Option) *Runtime

// RegisterSkill adds a skill and its tools to the MCP server.
func (r *Runtime) RegisterSkill(s skill.Skill) error

// WithAutoRegister enables automatic registration with omniskill registry.
func WithAutoRegister() Option
```

### Client (from mcpkit/client)

The existing `client/` package becomes `mcp/client/`:

```go
package client

// Client connects to external MCP servers.
type Client struct { ... }

// Session represents an active MCP connection.
type Session struct { ... }

// AsSkill wraps an MCP session as a skill.Skill.
func (s *Session) AsSkill(name string) (skill.Skill, error)
```

## Format Converters

### OpenAPI Importer

```go
package openapi

// Import generates a Skill from an OpenAPI specification.
func Import(spec []byte, opts ...Option) (skill.Skill, error)

// ImportURL fetches and imports an OpenAPI spec from a URL.
func ImportURL(ctx context.Context, url string, opts ...Option) (skill.Skill, error)
```

### Compiled Export

```go
package compiled

// Export converts a skill.Skill to omniagent's compiled.Skill format.
func Export(s skill.Skill) (omniagent.CompiledSkill, error)
```

## Migration from mcpkit

| mcpkit Path | omniskill Path | Notes |
|-------------|----------------|-------|
| `runtime/` | `mcp/server/` | MCP server runtime |
| `client/` | `mcp/client/` | MCP client |
| `oauth2/` | `mcp/oauth2/` | OAuth2 server |

### Import Path Changes

```go
// Before (mcpkit)
import "github.com/plexusone/mcpkit/runtime"
import "github.com/plexusone/mcpkit/client"
import "github.com/plexusone/mcpkit/oauth2"

// After (omniskill)
import "github.com/plexusone/omniskill/mcp/server"
import "github.com/plexusone/omniskill/mcp/client"
import "github.com/plexusone/omniskill/mcp/oauth2"
```

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/modelcontextprotocol/go-sdk` | v1.5.0+ | MCP protocol |
| `golang.ngrok.com/ngrok` | v1.13.0+ | Tunnel support |
| `github.com/grokify/mogo` | v0.74.0+ | Utilities |

## Testing Strategy

1. **Unit Tests** - Each package has `*_test.go` files
2. **Integration Tests** - MCP client ↔ server communication
3. **Format Tests** - Round-trip conversion (skill → format → skill)

## Performance Considerations

1. **Zero-Copy for Go Consumers** - Skills consumed directly as Go interfaces have no serialization overhead
2. **Lazy Tool Loading** - Tools discovered on demand for large registries
3. **Connection Pooling** - MCP client reuses connections

## Security Considerations

1. **Input Validation** - All tool parameters validated against schema
2. **OAuth2** - Token-based auth for MCP servers
3. **Subprocess Isolation** - External MCP servers run in separate processes
