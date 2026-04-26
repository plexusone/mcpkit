# Release Notes - v0.4.0

**Release Date:** 2026-03-07

## Overview

This release migrates MCPKit to the `plexusone` organization and updates the changelog format to use camelCase for ecosystem consistency with structured-changelog tooling.

## Installation

```bash
go get github.com/plexusone/mcpkit@v0.4.0
```

Requires Go 1.25+ and MCP Go SDK v1.3.0+.

## Highlights

- **Organization renamed** from `agentplexus` to `plexusone`
- **Changelog format** updated to camelCase for ecosystem consistency

## Breaking Changes

This release contains a breaking change to the module path.

### Import Path Change

```go
// Before (v0.3.x)
import "github.com/agentplexus/mcpkit/runtime"
import "github.com/agentplexus/mcpkit/oauth2"

// After (v0.4.0)
import "github.com/plexusone/mcpkit/runtime"
import "github.com/plexusone/mcpkit/oauth2"
```

## Upgrade Guide

### From v0.3.x

1. **Update go.mod**:
   ```bash
   go get github.com/plexusone/mcpkit@v0.4.0
   ```

2. **Update imports** in all Go files:
   - Replace `github.com/agentplexus/mcpkit` with `github.com/plexusone/mcpkit`

3. **Clean up dependencies**:
   ```bash
   go mod tidy
   ```

### Quick Migration Script

```bash
# In your project directory
find . -name "*.go" -exec sed -i '' \
  -e 's|github.com/agentplexus/mcpkit|github.com/plexusone/mcpkit|g' {} \;
go mod tidy
```

## What's Changed

### Changed

- All package imports updated to `github.com/plexusone/mcpkit`
- Documentation and badge URLs updated to plexusone organization
- Changelog JSON format migrated from snake_case to camelCase

## API Compatibility

All APIs remain functionally identical to v0.3.x. Only the import paths have changed:

| v0.3.x | v0.4.0 |
|--------|--------|
| `github.com/agentplexus/mcpkit/runtime` | `github.com/plexusone/mcpkit/runtime` |
| `github.com/agentplexus/mcpkit/oauth2` | `github.com/plexusone/mcpkit/oauth2` |

## Contributors

- John Wang

## Links

- [GitHub Repository](https://github.com/plexusone/mcpkit)
- [Go Package Documentation](https://pkg.go.dev/github.com/plexusone/mcpkit)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
