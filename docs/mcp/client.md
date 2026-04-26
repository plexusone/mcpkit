# MCP Client

The `mcp/client` package provides an MCP client for connecting to remote MCP servers. Remote servers can be wrapped as skills for unified tool access.

## Overview

The client package enables:

- Connecting to MCP servers via stdio or transport
- Listing and calling tools on remote servers
- Wrapping MCP sessions as skills

## Creating a Client

```go
import "github.com/plexusone/omniskill/mcp/client"

c := client.New("my-app", "1.0.0", nil)
```

With options:

```go
c := client.New("my-app", "1.0.0", &client.Options{
    Logger: slog.Default(),
})
```

## Connecting to Servers

### Via Command (stdio)

For MCP servers that run as subprocesses:

```go
import "os/exec"

cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp")
session, err := c.ConnectCommand(ctx, cmd)
if err != nil {
    log.Fatal(err)
}
defer session.Close()
```

### Via Transport

For custom transports:

```go
transport := createMyTransport()
session, err := c.Connect(ctx, transport)
```

## Session Operations

### List Tools

```go
tools, err := session.ListTools(ctx)
for _, tool := range tools {
    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
}
```

### Call Tool

```go
result, err := session.CallTool(ctx, "read_file", map[string]any{
    "path": "/tmp/example.txt",
})
if err != nil {
    log.Fatal(err)
}

// Access result content
for _, content := range result.Content {
    if tc, ok := content.(*mcp.TextContent); ok {
        fmt.Println(tc.Text)
    }
}
```

### List Prompts

```go
prompts, err := session.ListPrompts(ctx)
```

### Get Prompt

```go
result, err := session.GetPrompt(ctx, "greeting", map[string]string{
    "name": "Alice",
})
```

### List Resources

```go
resources, err := session.ListResources(ctx)
```

### Read Resource

```go
result, err := session.ReadResource(ctx, "file:///path/to/resource")
```

## Session as Skill

Wrap an MCP session as a skill for unified access:

```go
// Create skill from session
skill := session.AsSkill(
    client.WithSkillName("filesystem"),
    client.WithSkillDescription("File system operations"),
)

// Use like any other skill
fmt.Println("Skill:", skill.Name())

for _, tool := range skill.Tools() {
    fmt.Printf("Tool: %s - %s\n", tool.Name(), tool.Description())
}

// Call tools through skill interface
result, err := skill.Tools()[0].Call(ctx, map[string]any{
    "path": "/tmp/test.txt",
})
```

### Skill Options

```go
session.AsSkill(
    client.WithSkillName("github"),       // Custom skill name
    client.WithSkillDescription("..."),   // Custom description
)
```

### Lazy Initialization

Tools are discovered lazily on first access:

```go
skill := session.AsSkill()

// First call discovers tools from server
tools := skill.Tools()

// Subsequent calls use cached tools
tools = skill.Tools()  // No server call
```

Or initialize explicitly:

```go
skill := session.AsSkill()
if err := skill.Init(ctx); err != nil {
    log.Fatal(err)
}
```

## Session Lifecycle

```go
session, err := c.ConnectCommand(ctx, cmd)
if err != nil {
    log.Fatal(err)
}

// Use session...

// Clean up
session.Close()

// Or wait for server to close
session.Wait()
```

## Session Information

```go
// Session ID
id := session.ID()

// Server capabilities
initResult := session.InitializeResult()
fmt.Println("Server:", initResult.ServerInfo.Name)

// Access underlying MCP session
mcpSession := session.MCPSession()
```

## Example: Using Multiple MCP Servers

```go
// Connect to multiple servers
fsSession, _ := c.ConnectCommand(ctx, exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"))
defer fsSession.Close()

githubSession, _ := c.ConnectCommand(ctx, exec.Command("npx", "-y", "@modelcontextprotocol/server-github"))
defer githubSession.Close()

// Wrap as skills
fsSkill := fsSession.AsSkill(client.WithSkillName("filesystem"))
githubSkill := githubSession.AsSkill(client.WithSkillName("github"))

// Register with registry
reg := registry.New()
reg.Register(fsSkill)
reg.Register(githubSkill)

// Now all tools from both servers are discoverable
for _, tool := range reg.ListTools() {
    fmt.Println(tool.Name())
}
```

## Example: With MCP Server Runtime

```go
// Create runtime
rt := server.New(impl, nil)

// Connect to remote server
session, _ := c.ConnectCommand(ctx, cmd)
remoteSkill := session.AsSkill(client.WithSkillName("remote"))

// Register remote skill with runtime
rt.RegisterSkillWithPrefix(remoteSkill)

// Tools are now available as "remote_toolname"
rt.ServeStdio(ctx)
```
