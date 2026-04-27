# Skill Packs

Skill packs are distributable bundles of markdown skills embedded via Go's `embed` package. They implement the `pack.SkillPack` interface, allowing skills to be shared as Go modules.

## Overview

```
┌─────────────────────────────────────────────┐
│              Skill Pack Module              │
│  ┌─────────────────────────────────────┐   │
│  │          //go:embed skills/*        │   │
│  │              skillsFS               │   │
│  └─────────────────────────────────────┘   │
│                    │                        │
│  ┌─────────────────┴─────────────────┐     │
│  │         pack.SkillPack            │     │
│  │   Name() | Version() | FS()       │     │
│  └───────────────────────────────────┘     │
└─────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│              Agent Runtime                  │
│         agent.WithSkillPack(fs.FS)          │
└─────────────────────────────────────────────┘
```

## The SkillPack Interface

```go
package pack

import "embed"

// SkillPack defines the interface for markdown skill bundles.
type SkillPack interface {
    // Name returns the pack identifier (e.g., "omniagent-skills").
    Name() string

    // Version returns the pack version (e.g., commit hash or semver).
    Version() string

    // FS returns the embedded filesystem containing skills.
    // Skills should be in a "skills/" subdirectory.
    FS() embed.FS
}
```

## Creating a Skill Pack

### 1. Project Structure

```
my-skill-pack/
├── go.mod
├── pack.go           # SkillPack implementation
├── VERSION           # Version file (optional)
└── skills/
    ├── weather/
    │   └── SKILL.md
    ├── github/
    │   └── SKILL.md
    └── tmux/
        └── SKILL.md
```

### 2. Implement SkillPack

```go
package myskills

import (
    "embed"
    "strings"

    "github.com/plexusone/omniskill/pack"
)

//go:embed skills/*
var skillsFS embed.FS

//go:embed VERSION
var version string

// Pack implements pack.SkillPack for this skill bundle.
type Pack struct{}

func (Pack) Name() string {
    return "my-skill-pack"
}

func (Pack) Version() string {
    return strings.TrimSpace(version)
}

func (Pack) FS() embed.FS {
    return skillsFS
}

// Default returns a new Pack instance.
func Default() *Pack {
    return &Pack{}
}

// Ensure Pack implements pack.SkillPack
var _ pack.SkillPack = (*Pack)(nil)
```

### 3. Add Skills

Create `SKILL.md` files in the `skills/` directory:

```markdown
---
name: weather
description: Get weather forecasts
metadata:
  emoji: "🌤️"
  requires:
    bins: ["curl"]
---

# Weather Skill

Check the weather using curl:

## Current Weather

```bash
curl "wttr.in/London?format=3"
```
```

### 4. Version File

Create a `VERSION` file with the current version:

```
v1.0.0
```

Or use a git commit hash during build:

```bash
git rev-parse HEAD > VERSION
```

## Using a Skill Pack

### With OmniAgent

```go
import (
    "github.com/plexusone/omniagent/agent"
    skills "github.com/example/my-skill-pack"
)

agent, err := agent.New(config,
    agent.WithSkillPack(skills.Default().FS()),
)
```

### With Filtering

```go
agent, err := agent.New(config,
    agent.WithSkillPack(skills.Default().FS()),
    agent.WithSkillIncludes("weather", "github"),
)
```

### Multiple Packs

```go
import (
    defaultSkills "github.com/plexusone/omniagent-skills"
    customSkills "github.com/example/my-skill-pack"
)

agent, err := agent.New(config,
    agent.WithSkillPack(defaultSkills.Default().FS()),
    agent.WithSkillPack(customSkills.Default().FS()),
)
```

## Directory Structure Requirements

The embedded filesystem must have skills in a `skills/` subdirectory:

```
skills/
├── skill-name/
│   └── SKILL.md
├── another-skill/
│   └── SKILL.md
```

Each skill directory must contain a `SKILL.md` file with YAML frontmatter.

## Best Practices

### 1. Use Semantic Versioning

Tag releases with semantic versions (v1.0.0, v1.1.0, etc.) so consumers can pin specific versions.

### 2. Document Requirements

Each skill should declare its requirements in the YAML frontmatter:

```yaml
metadata:
  requires:
    bins: ["curl", "jq"]
    env: ["API_KEY"]
```

### 3. Include Install Hints

Help users install missing dependencies:

```yaml
metadata:
  install:
    - name: curl
      brew: curl
      apt: curl
```

### 4. Test Skills

Verify skills load correctly:

```go
func TestPackFS(t *testing.T) {
    p := Default()
    fsys := p.FS()

    entries, err := fs.ReadDir(fsys, "skills")
    if err != nil {
        t.Fatalf("ReadDir failed: %v", err)
    }

    if len(entries) == 0 {
        t.Error("No skills found")
    }
}
```

## Available Packs

| Pack | Description | Install |
|------|-------------|---------|
| [omniagent-skills](https://github.com/plexusone/omniagent-skills) | Default pack with 18 skills | `go get github.com/plexusone/omniagent-skills` |

## See Also

- [Skills](skills.md) - Creating individual skills
- [Registry](registry.md) - Skill registration and discovery
