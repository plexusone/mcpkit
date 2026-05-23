# Tools

Tools are individual functions that AI agents can invoke. Each tool has a name, description, parameters, and a handler function.

## Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]Parameter
    Call(ctx context.Context, params map[string]any) (any, error)
}
```

## Creating Tools

Use `skill.NewTool` to create tools:

```go
import "github.com/plexusone/omniskill/skill"

addTool := skill.NewTool(
    "add",                    // Name
    "Add two numbers",        // Description
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
```

## Parameters

Parameters define the input schema for tools. They map to JSON Schema.

### Parameter Fields

```go
type Parameter struct {
    Type        string                 // JSON Schema type
    Description string                 // Human-readable description
    Required    bool                   // Is this parameter required?
    Enum        []any                  // Allowed values
    Default     any                    // Default value
    Items       *Parameter             // For array types
    Properties  map[string]Parameter   // For object types
}
```

### Basic Types

```go
// String
"name": {Type: "string", Description: "User's name", Required: true}

// Number (float64)
"amount": {Type: "number", Description: "Amount in dollars"}

// Integer
"count": {Type: "integer", Description: "Number of items", Default: 10}

// Boolean
"verbose": {Type: "boolean", Description: "Enable verbose output"}
```

### Enum Values

```go
"format": {
    Type: "string",
    Description: "Output format",
    Enum: []any{"json", "xml", "csv"},
    Default: "json",
}
```

### Arrays

```go
"tags": {
    Type: "array",
    Description: "List of tags",
    Items: &skill.Parameter{Type: "string"},
}
```

### Nested Objects

```go
"address": {
    Type: "object",
    Description: "Mailing address",
    Properties: map[string]skill.Parameter{
        "street": {Type: "string", Required: true},
        "city":   {Type: "string", Required: true},
        "zip":    {Type: "string"},
    },
}
```

## Handler Functions

Handlers receive context and parameters, returning a result or error:

```go
func(ctx context.Context, params map[string]any) (any, error)
```

### Accessing Parameters

Parameters are passed as `map[string]any`. Type assert to access values:

```go
func(ctx context.Context, params map[string]any) (any, error) {
    // Required string
    name := params["name"].(string)

    // Optional with default
    count := 10
    if c, ok := params["count"]; ok {
        count = int(c.(float64))  // JSON numbers are float64
    }

    // Array
    if tags, ok := params["tags"].([]any); ok {
        for _, tag := range tags {
            fmt.Println(tag.(string))
        }
    }

    return map[string]any{"processed": name, "count": count}, nil
}
```

### Returning Results

Return any JSON-serializable value:

```go
// Map
return map[string]any{"status": "ok", "count": 42}, nil

// String
return "Operation completed", nil

// Struct (will be JSON-marshaled)
return MyResult{Status: "ok"}, nil
```

### Returning Errors

Return errors to indicate failures:

```go
func(ctx context.Context, params map[string]any) (any, error) {
    filename := params["filename"].(string)

    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    return string(data), nil
}
```

## CommandTool

`CommandTool` wraps CLI commands as Tool implementations. This enables SKILL.md-defined commands to be exposed as standard tools.

```go
import "github.com/plexusone/omniskill/skill"

tool := &skill.CommandTool{
    ToolName:        "search",
    ToolDescription: "Search the archive",
    Command:         "notcrawl",
    Args:            []string{"search", "{{query}}"},
    ToolParameters: map[string]skill.Parameter{
        "query": {Type: "string", Description: "Search query", Required: true},
    },
    Timeout: 30 * time.Second,
}

// Call the tool
result, err := tool.Call(ctx, map[string]any{"query": "meeting notes"})

// Result is a CommandResult
cmdResult := result.(skill.CommandResult)
fmt.Println(cmdResult.Stdout)
fmt.Printf("Exit code: %d\n", cmdResult.ExitCode)
```

### CommandTool Fields

| Field | Description |
|-------|-------------|
| `ToolName` | Tool identifier |
| `ToolDescription` | Human-readable description |
| `Command` | Executable name or path |
| `Args` | Command arguments (supports `{{param}}` substitution) |
| `ToolParameters` | Input parameter definitions |
| `WorkingDir` | Working directory (default: current directory) |
| `Timeout` | Maximum execution time (zero = no timeout) |
| `Env` | Additional environment variables |

### Parameter Substitution

Use `{{paramName}}` placeholders in `Args` to inject parameter values:

```go
tool := &skill.CommandTool{
    Command: "gh",
    Args:    []string{"issue", "view", "{{number}}", "--repo", "{{repo}}"},
    ToolParameters: map[string]skill.Parameter{
        "number": {Type: "integer", Required: true},
        "repo":   {Type: "string", Required: true},
    },
}

// Calling with {"number": 123, "repo": "user/repo"} executes:
// gh issue view 123 --repo user/repo
```

### CommandResult

```go
type CommandResult struct {
    Stdout   string `json:"stdout"`   // Standard output
    Stderr   string `json:"stderr"`   // Standard error
    ExitCode int    `json:"exit_code"` // Exit code (0 = success)
}

// String() returns stdout on success, stderr on failure
fmt.Println(result.String())
```

### Using NewCommandTool

Helper function with common defaults (30 second timeout):

```go
tool := skill.NewCommandTool(
    "list_files",
    "List files in a directory",
    "ls",
    []string{"-la", "{{path}}"},
    map[string]skill.Parameter{
        "path": {Type: "string", Description: "Directory path", Required: true},
    },
)
```

### Environment Variables

Pass additional environment variables to commands:

```go
tool := &skill.CommandTool{
    Command: "my-tool",
    Args:    []string{"run"},
    Env:     []string{"API_KEY=secret", "DEBUG=true"},
}
```

## MCP Tool Conversion

When skills are registered with an MCP server runtime, tools are automatically converted to MCP format:

```go
// skill.Tool → mcp.Tool
// Parameters → JSON Schema (InputSchema)
// Handler → mcp.ToolHandler
```

The conversion handles:

- Parameter types to JSON Schema types
- Required fields to `required` array
- Nested objects and arrays
- Default values and enums

## Best Practices

### 1. Clear Names and Descriptions

Names should be verb-noun format. Descriptions should explain what the tool does:

```go
// Good
skill.NewTool("get_weather", "Get current weather conditions for a location", ...)
skill.NewTool("send_email", "Send an email message to recipients", ...)

// Bad
skill.NewTool("weather", "Weather", ...)  // Too vague
skill.NewTool("gw", "Gets weather", ...)  // Cryptic name
```

### 2. Validate Required Parameters

Check for required parameters before using them:

```go
func(ctx context.Context, params map[string]any) (any, error) {
    name, ok := params["name"].(string)
    if !ok || name == "" {
        return nil, errors.New("name is required")
    }
    // ...
}
```

### 3. Use Context

Respect context cancellation for long operations:

```go
func(ctx context.Context, params map[string]any) (any, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Long operation...
    return result, nil
}
```

### 4. Return Structured Results

Return maps or structs for rich results:

```go
// Good - structured
return map[string]any{
    "success": true,
    "data": processedData,
    "metadata": map[string]any{
        "processed_at": time.Now(),
        "count": len(processedData),
    },
}, nil

// Less useful - just a string
return "done", nil
```
