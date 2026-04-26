# API Reference

Full API documentation is available on pkg.go.dev:

- [github.com/plexusone/omniskill](https://pkg.go.dev/github.com/plexusone/omniskill)
- [github.com/plexusone/omniskill/skill](https://pkg.go.dev/github.com/plexusone/omniskill/skill)
- [github.com/plexusone/omniskill/registry](https://pkg.go.dev/github.com/plexusone/omniskill/registry)
- [github.com/plexusone/omniskill/mcp/server](https://pkg.go.dev/github.com/plexusone/omniskill/mcp/server)
- [github.com/plexusone/omniskill/mcp/client](https://pkg.go.dev/github.com/plexusone/omniskill/mcp/client)
- [github.com/plexusone/omniskill/mcp/oauth2](https://pkg.go.dev/github.com/plexusone/omniskill/mcp/oauth2)

## Quick Reference

### skill Package

```go
// Core interfaces
type Skill interface {
    Name() string
    Description() string
    Tools() []Tool
    Init(ctx context.Context) error
    Close() error
}

type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]Parameter
    Call(ctx context.Context, params map[string]any) (any, error)
}

// Parameter definition
type Parameter struct {
    Type        string
    Description string
    Required    bool
    Enum        []any
    Default     any
    Items       *Parameter
    Properties  map[string]Parameter
}

// Convenience types
type BaseSkill struct { ... }
func NewTool(name, description string, params map[string]Parameter, handler ToolFunc) *FuncTool
```

### registry Package

```go
type Registry interface {
    Register(s skill.Skill) error
    Unregister(name string) error
    Get(name string) (skill.Skill, error)
    List() []skill.Skill
    ListTools() []skill.Tool
    GetTool(fullName string) (skill.Tool, error)
    Init(ctx context.Context) error
    Close() error
}

func New() *InMemory

var ErrSkillNotFound = errors.New("skill not found")
var ErrSkillExists = errors.New("skill already registered")
```

### mcp/server Package

```go
// Runtime creation
func New(impl *mcp.Implementation, opts *Options) *Runtime

type Options struct {
    Logger        *slog.Logger
    ServerOptions *mcp.ServerOptions
    Registry      registry.Registry
}

// Skill registration
func (r *Runtime) RegisterSkill(s skill.Skill)
func (r *Runtime) RegisterSkillWithPrefix(s skill.Skill)

// Tool registration (MCP-style)
func AddTool[In, Out any](r *Runtime, tool *mcp.Tool, handler ToolHandlerFor[In, Out])
func (r *Runtime) AddToolHandler(tool *mcp.Tool, handler mcp.ToolHandler)

// Library mode
func (r *Runtime) CallTool(ctx context.Context, name string, args any) (*mcp.CallToolResult, error)
func (r *Runtime) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error)
func (r *Runtime) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)

// Server mode
func (r *Runtime) ServeStdio(ctx context.Context) error
func (r *Runtime) ServeHTTP(ctx context.Context, opts *HTTPServerOptions) (*HTTPServerResult, error)
func (r *Runtime) StreamableHTTPHandler(opts *StreamableHTTPOptions) http.Handler
func (r *Runtime) SSEHandler(opts *SSEOptions) http.Handler

// Inspection
func (r *Runtime) ListTools() []*mcp.Tool
func (r *Runtime) HasTool(name string) bool
func (r *Runtime) ToolCount() int
```

### mcp/client Package

```go
// Client creation
func New(name, version string, opts *Options) *Client

// Connection
func (c *Client) Connect(ctx context.Context, transport mcp.Transport) (*Session, error)
func (c *Client) ConnectCommand(ctx context.Context, cmd *exec.Cmd) (*Session, error)

// Session operations
func (s *Session) ListTools(ctx context.Context) ([]*mcp.Tool, error)
func (s *Session) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error)
func (s *Session) ListPrompts(ctx context.Context) ([]*mcp.Prompt, error)
func (s *Session) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error)
func (s *Session) ListResources(ctx context.Context) ([]*mcp.Resource, error)
func (s *Session) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error)
func (s *Session) Close() error

// Session as skill
func (s *Session) AsSkill(opts ...SkillOption) *SessionSkill
func WithSkillName(name string) SkillOption
func WithSkillDescription(desc string) SkillOption
```
