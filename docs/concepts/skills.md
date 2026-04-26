# Skills

Skills are the primary abstraction in OmniSkill. A skill is a named collection of related tools with lifecycle management.

## Skill Interface

```go
type Skill interface {
    Name() string
    Description() string
    Tools() []Tool
    Init(ctx context.Context) error
    Close() error
}
```

| Method | Purpose |
|--------|---------|
| `Name()` | Returns the skill's unique identifier |
| `Description()` | Human-readable description |
| `Tools()` | Returns all tools provided by this skill |
| `Init(ctx)` | Initialize the skill (connect to services, load resources) |
| `Close()` | Clean up resources |

## Using BaseSkill

For simple skills, use the provided `BaseSkill` struct:

```go
import "github.com/plexusone/omniskill/skill"

mySkill := &skill.BaseSkill{
    SkillName:        "calculator",
    SkillDescription: "Mathematical operations",
    SkillTools: []skill.Tool{
        skill.NewTool("add", "Add numbers", params, handler),
        skill.NewTool("multiply", "Multiply numbers", params, handler),
    },
}
```

`BaseSkill` provides no-op implementations of `Init()` and `Close()`.

## Custom Skills

For skills that need initialization or cleanup, implement the interface directly:

```go
type WeatherSkill struct {
    apiKey string
    client *http.Client
}

func (s *WeatherSkill) Name() string {
    return "weather"
}

func (s *WeatherSkill) Description() string {
    return "Weather forecasts and conditions"
}

func (s *WeatherSkill) Tools() []skill.Tool {
    return []skill.Tool{
        skill.NewTool("current", "Get current weather",
            map[string]skill.Parameter{
                "location": {Type: "string", Required: true},
            },
            s.getCurrentWeather,
        ),
        skill.NewTool("forecast", "Get weather forecast",
            map[string]skill.Parameter{
                "location": {Type: "string", Required: true},
                "days":     {Type: "integer", Default: 5},
            },
            s.getForecast,
        ),
    }
}

func (s *WeatherSkill) Init(ctx context.Context) error {
    // Validate API key, create HTTP client, etc.
    s.client = &http.Client{Timeout: 10 * time.Second}
    return nil
}

func (s *WeatherSkill) Close() error {
    s.client.CloseIdleConnections()
    return nil
}

func (s *WeatherSkill) getCurrentWeather(ctx context.Context, params map[string]any) (any, error) {
    location := params["location"].(string)
    // Call weather API...
    return map[string]any{"temp": 72, "condition": "sunny"}, nil
}

func (s *WeatherSkill) getForecast(ctx context.Context, params map[string]any) (any, error) {
    // Implementation...
    return nil, nil
}
```

## Skill Lifecycle

Skills follow this lifecycle:

```
┌──────────┐     Init()     ┌──────────┐     Close()    ┌──────────┐
│  Created │ ────────────▶  │  Ready   │ ────────────▶  │  Closed  │
└──────────┘                └──────────┘                └──────────┘
                                  │
                                  │ Tools available
                                  ▼
                            ┌──────────┐
                            │  In Use  │
                            └──────────┘
```

1. **Created** - Skill instance exists but not initialized
2. **Ready** - `Init()` called successfully, tools can be invoked
3. **Closed** - `Close()` called, resources released

## Registering Skills

### With MCP Server Runtime

```go
rt := server.New(impl, nil)
rt.RegisterSkill(mySkill)

// Or with tool name prefixing
rt.RegisterSkillWithPrefix(mySkill) // Tools become "skillname_toolname"
```

### With Auto-Registration

```go
reg := registry.New()
rt := server.New(impl, &server.Options{
    Registry: reg,  // Skills auto-register here
})

rt.RegisterSkill(mySkill)

// Skill is now in both the runtime AND the registry
```

### With Registry Only

```go
reg := registry.New()
reg.Register(mySkill)

// Initialize all registered skills
if err := reg.Init(ctx); err != nil {
    log.Fatal(err)
}
```

## MCP Sessions as Skills

Remote MCP servers can be wrapped as skills:

```go
// Connect to MCP server
session, err := client.ConnectCommand(ctx, cmd)

// Wrap as skill
remoteSkill := session.AsSkill(
    client.WithSkillName("github"),
    client.WithSkillDescription("GitHub operations"),
)

// Use like any other skill
for _, tool := range remoteSkill.Tools() {
    fmt.Println(tool.Name())
}
```

## Best Practices

### 1. Group Related Tools

A skill should contain tools that logically belong together:

```go
// Good: related tools
fileSkill := &skill.BaseSkill{
    SkillName: "files",
    SkillTools: []skill.Tool{
        skill.NewTool("read", ...),
        skill.NewTool("write", ...),
        skill.NewTool("delete", ...),
    },
}

// Bad: unrelated tools
mixedSkill := &skill.BaseSkill{
    SkillName: "misc",
    SkillTools: []skill.Tool{
        skill.NewTool("read_file", ...),
        skill.NewTool("send_email", ...),  // Different domain
        skill.NewTool("query_database", ...), // Different domain
    },
}
```

### 2. Handle Initialization Errors

Return errors from `Init()` to prevent use of uninitialized skills:

```go
func (s *DatabaseSkill) Init(ctx context.Context) error {
    db, err := sql.Open("postgres", s.connString)
    if err != nil {
        return fmt.Errorf("failed to connect to database: %w", err)
    }
    s.db = db
    return nil
}
```

### 3. Clean Up Resources

Always release resources in `Close()`:

```go
func (s *DatabaseSkill) Close() error {
    if s.db != nil {
        return s.db.Close()
    }
    return nil
}
```

### 4. Use Descriptive Names

Skill and tool names should be clear and consistent:

```go
// Good
SkillName: "weather"
tool: "get_current_conditions"
tool: "get_forecast"

// Bad
SkillName: "w"
tool: "gc"
tool: "f"
```
