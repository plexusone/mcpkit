# Installation

## Requirements

- Go 1.21 or later
- Git

## Install

```bash
go get github.com/plexusone/omniskill
```

## Import Packages

Import the packages you need:

```go
import (
    // Core skill interfaces
    "github.com/plexusone/omniskill/skill"

    // Skill registry
    "github.com/plexusone/omniskill/registry"

    // MCP server runtime
    "github.com/plexusone/omniskill/mcp/server"

    // MCP client
    "github.com/plexusone/omniskill/mcp/client"

    // OAuth2 (for authenticated MCP servers)
    "github.com/plexusone/omniskill/mcp/oauth2"
)
```

## Verify Installation

Create a simple test file:

```go
package main

import (
    "fmt"
    "github.com/plexusone/omniskill/skill"
)

func main() {
    s := &skill.BaseSkill{
        SkillName: "test",
    }
    fmt.Println("Skill name:", s.Name())
}
```

Run it:

```bash
go run main.go
# Output: Skill name: test
```

## Next Steps

- [Quick Start](quickstart.md) - Build your first skill
- [Concepts Overview](../concepts/overview.md) - Understand the architecture
