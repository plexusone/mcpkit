# Registry

The Registry provides central skill registration and discovery. It manages the lifecycle of skills and enables tool lookup across all registered skills.

## Registry Interface

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
    Count() int
}
```

## Creating a Registry

```go
import "github.com/plexusone/omniskill/registry"

reg := registry.New()
```

## Registering Skills

```go
// Register a skill
err := reg.Register(&skill.BaseSkill{
    SkillName:        "math",
    SkillDescription: "Mathematical operations",
    SkillTools:       []skill.Tool{addTool, multiplyTool},
})

if errors.Is(err, registry.ErrSkillExists) {
    // Skill with this name already registered
}
```

## Retrieving Skills

```go
// Get by name
mathSkill, err := reg.Get("math")
if errors.Is(err, registry.ErrSkillNotFound) {
    // Skill not found
}

// List all skills
for _, s := range reg.List() {
    fmt.Printf("Skill: %s - %s\n", s.Name(), s.Description())
}

// Count registered skills
count := reg.Count()
```

## Tool Discovery

The registry enables tool discovery across all skills:

```go
// List all tools from all skills
tools := reg.ListTools()
for _, t := range tools {
    fmt.Printf("Tool: %s\n", t.Name())
}

// Get tool by full name (skill.tool)
tool, err := reg.GetTool("math.add")

// Get tool by short name (searches all skills)
tool, err := reg.GetTool("add")
```

## Lifecycle Management

The registry manages skill initialization and cleanup:

```go
// Initialize all registered skills
if err := reg.Init(ctx); err != nil {
    log.Fatal("Failed to initialize skills:", err)
}

// Later, clean up all skills
if err := reg.Close(); err != nil {
    log.Println("Error closing skills:", err)
}
```

Init stops at the first error. Close continues through all skills and returns a joined error if any fail.

## Auto-Registration with MCP Server

Skills registered with an MCP server runtime can automatically register with a registry:

```go
reg := registry.New()

rt := server.New(impl, &server.Options{
    Registry: reg,  // Enable auto-registration
})

// This registers with BOTH the runtime AND the registry
rt.RegisterSkill(mySkill)

// Verify it's in the registry
s, _ := reg.Get("myskill")
fmt.Println(s.Name())  // "myskill"
```

## Unregistering Skills

```go
err := reg.Unregister("math")
if errors.Is(err, registry.ErrSkillNotFound) {
    // Skill was not registered
}
```

## Thread Safety

The in-memory registry is thread-safe. Multiple goroutines can safely register, unregister, and query skills concurrently.

```go
// Safe to call from multiple goroutines
go func() { reg.Register(skill1) }()
go func() { reg.Register(skill2) }()
go func() { reg.List() }()
```

## Error Handling

```go
var (
    ErrSkillNotFound = errors.New("skill not found")
    ErrSkillExists   = errors.New("skill already registered")
)
```

Use `errors.Is()` to check error types:

```go
err := reg.Register(skill)
if errors.Is(err, registry.ErrSkillExists) {
    // Handle duplicate registration
}
```

## Use Cases

### 1. Agent Tool Discovery

```go
// Agent discovers all available tools
tools := reg.ListTools()
toolDescriptions := make([]string, len(tools))
for i, t := range tools {
    toolDescriptions[i] = fmt.Sprintf("%s: %s", t.Name(), t.Description())
}
// Send to LLM as available tools
```

### 2. Dynamic Skill Loading

```go
// Load skills based on configuration
for _, cfg := range config.Skills {
    skill := createSkillFromConfig(cfg)
    reg.Register(skill)
}

// Initialize all at once
reg.Init(ctx)
```

### 3. Graceful Shutdown

```go
// On shutdown signal
sig := make(chan os.Signal, 1)
signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
<-sig

// Close all skills
if err := reg.Close(); err != nil {
    log.Printf("Warning: error closing skills: %v", err)
}
```

### 4. Skill Hot-Reloading

```go
// Remove old version
reg.Unregister("weather")

// Register new version
reg.Register(newWeatherSkill)
newWeatherSkill.Init(ctx)
```
