# Release Notes - v0.3.0

**Release Date:** 2026-01-18

## Overview

This release marks the transition of the project to its new home as **MCPKit** under `github.com/plexusone/mcpkit`. The project has been restructured into focused subpackages for better organization and future growth:

- **`runtime/`** - Core MCP server runtime with tools, prompts, resources, and transports
- **`oauth2/`** - OAuth 2.1 Authorization Server with PKCE support

This release also includes error handling improvements following Go best practices.

## Installation

```bash
go get github.com/plexusone/mcpkit@v0.3.0
```

Requires Go 1.24+ and MCP Go SDK v1.2.0+.

## Highlights

- **Project renamed** to MCPKit under `github.com/plexusone/mcpkit`
- **Restructured** into focused subpackages: `runtime/` and `oauth2/`
- **Improved error handling** following Go best practices

## Breaking Changes

This release contains breaking changes that require updates to your import statements and package references.

### Import Path Changes

```go
// Before (v0.2.0)
import "github.com/grokify/mcpruntime"
import "github.com/grokify/mcpruntime/oauth2server"

// After (v0.3.0)
import "github.com/plexusone/mcpkit/runtime"
import "github.com/plexusone/mcpkit/oauth2"
```

### Package Name Changes

```go
// Before (v0.2.0)
rt := mcpruntime.New(&mcp.Implementation{...}, nil)
mcpruntime.AddTool(rt, tool, handler)

// After (v0.3.0)
rt := runtime.New(&mcp.Implementation{...}, nil)
runtime.AddTool(rt, tool, handler)
```

### OAuth2 Server Changes

```go
// Before (v0.2.0)
import "github.com/grokify/mcpruntime/oauth2server"
srv, err := oauth2server.New(&oauth2server.Config{...})

// After (v0.3.0)
import "github.com/plexusone/mcpkit/oauth2"
srv, err := oauth2.New(&oauth2.Config{...})
```

## Upgrade Guide

### From v0.2.0

1. **Update go.mod**:
   ```bash
   go get github.com/plexusone/mcpkit@v0.3.0
   ```

2. **Update imports** in all Go files:
   - Replace `github.com/grokify/mcpruntime` with `github.com/plexusone/mcpkit/runtime`
   - Replace `github.com/grokify/mcpruntime/oauth2server` with `github.com/plexusone/mcpkit/oauth2`

3. **Update package references**:
   - Replace `mcpruntime.` with `runtime.`
   - Replace `oauth2server.` with `oauth2.`

4. **Remove old dependency**:
   ```bash
   go mod tidy
   ```

### Quick Migration Script

```bash
# In your project directory
find . -name "*.go" -exec sed -i '' \
  -e 's|github.com/grokify/mcpruntime/oauth2server|github.com/plexusone/mcpkit/oauth2|g' \
  -e 's|github.com/grokify/mcpruntime|github.com/plexusone/mcpkit/runtime|g' \
  -e 's|mcpruntime\.|runtime.|g' \
  -e 's|oauth2server\.|oauth2.|g' {} \;
go mod tidy
```

## What's Changed

### Changed

- Module path changed from `github.com/grokify/mcpruntime` to `github.com/plexusone/mcpkit`
- Core runtime code moved to `runtime/` subpackage
- OAuth2 server moved from `oauth2server/` to `oauth2/`
- Package references changed from `mcpkit.` to `runtime.` for runtime types

### Fixed

- Error handling for `fmt.Fprintf` in OAuth login error page now logs errors via `slog.Logger`
- Error handling for `resp.Body.Close()` in all test files now reports errors via `t.Logf`

## Package Structure

```
github.com/plexusone/mcpkit
├── runtime/     # Core MCP server runtime
│   ├── Runtime type (New, CallTool, ServeStdio, ServeHTTP, etc.)
│   ├── Tool, Prompt, Resource registration
│   └── OAuth options for HTTP serving
├── oauth2/      # OAuth 2.1 Authorization Server
│   ├── Authorization Code Flow with PKCE (RFC 7636)
│   ├── Dynamic Client Registration (RFC 7591)
│   └── Authorization Server Metadata (RFC 8414)
└── doc.go       # Package documentation
```

## API Compatibility

All APIs remain functionally identical to v0.2.0. Only the import paths and package names have changed:

| v0.2.0 | v0.3.0 |
|--------|--------|
| `mcpruntime.New()` | `runtime.New()` |
| `mcpruntime.AddTool()` | `runtime.AddTool()` |
| `mcpruntime.Options` | `runtime.Options` |
| `mcpruntime.HTTPServerOptions` | `runtime.HTTPServerOptions` |
| `mcpruntime.OAuth2Options` | `runtime.OAuth2Options` |
| `oauth2server.New()` | `oauth2.New()` |
| `oauth2server.Config` | `oauth2.Config` |

## Contributors

- John Wang

## Links

- [GitHub Repository](https://github.com/plexusone/mcpkit)
- [Go Package Documentation](https://pkg.go.dev/github.com/plexusone/mcpkit)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
