# Loader

The `loader` package provides a unified interface for loading skills from various formats, enabling "define once, deploy everywhere" across different skill definition types.

## Skill Formats

OmniSkill supports multiple skill definition formats:

| Format | Description | Use Case |
|--------|-------------|----------|
| `markdown` | OpenClaw SKILL.md files | Declarative CLI wrapper skills |
| `go` | Native Go implementations | Full programmatic control |
| `mcp_server` | MCP server processes | Remote tool integration |
| `openapi` | OpenAPI specifications | REST API skills (planned) |

## Loading SKILL.md Files

The most common use case is loading SKILL.md markdown skills:

```go
import "github.com/plexusone/omniskill/loader"

// Load from file path
skill, err := loader.LoadMarkdownSkill("./skills/notcrawl/SKILL.md")

// Load from directory (looks for SKILL.md inside)
skill, err := loader.LoadMarkdownSkillDir("./skills/notcrawl")
```

## SKILL.md Format

SKILL.md files use YAML frontmatter with markdown body:

```markdown
---
name: notcrawl
description: "Notion archive search and sync"
metadata:
  openclaw:
    homepage: https://github.com/user/notcrawl
    requires:
      bins: [notcrawl]
    install:
      - kind: go
        module: github.com/user/notcrawl@latest
        bins: [notcrawl]
---
# Usage

Search your Notion archive:

```bash
notcrawl search "meeting notes"
```

Sync the archive:

```bash
notcrawl sync --force
```
```

### Frontmatter Fields

| Field | Description |
|-------|-------------|
| `name` | Skill identifier |
| `description` | Human-readable description |
| `metadata.openclaw.homepage` | Project homepage URL |
| `metadata.openclaw.requires.bins` | Required binary executables |
| `metadata.openclaw.install` | Installation instructions |

### Install Steps

Each install step specifies how to install a dependency:

```yaml
install:
  - kind: go
    module: github.com/user/tool@latest
    bins: [tool]
    script: "tool setup"  # Optional post-install script
```

Supported kinds: `go`, `npm`, `pip`, `docker`, `brew`

## MarkdownSkill

`MarkdownSkill` implements `skill.Skill` for loaded SKILL.md definitions:

```go
skill, err := loader.LoadMarkdownSkill("SKILL.md")

// Access metadata
fmt.Println(skill.Name())        // "notcrawl"
fmt.Println(skill.Description()) // "Notion archive search and sync"

// Get the markdown guidance (for AI context)
guidance := skill.GetGuidance()

// Get installation steps
steps := skill.GetInstallSteps()

// Use tools discovered from code blocks
for _, tool := range skill.Tools() {
    fmt.Printf("%s: %s\n", tool.Name(), tool.Description())
}
```

### Automatic Tool Discovery

The parser discovers commands from markdown code blocks and generates tools:

1. **General `run` tool** - For each required binary, a `run` tool allows arbitrary command execution
2. **Specific tools** - Commands with subcommands become named tools (e.g., `notcrawl search` → `notcrawl_search`)

Tools use `CommandTool` internally (see [Tools](tools.md#commandtool)).

## Discovering Skills

Find all skills in a directory:

```go
// Discover skills in a directory
infos, err := loader.DiscoverSkills("./skills")

for _, info := range infos {
    fmt.Printf("Found: %s (%v)\n", info.Name, info.Formats)
}
```

### SkillInfo

`SkillInfo` describes a discovered skill:

```go
type SkillInfo struct {
    Dir          string        // Directory path
    Name         string        // Skill name
    Formats      []SkillFormat // Available formats (markdown, go)
    MarkdownPath string        // Path to SKILL.md if present
    GoPath       string        // Path to skill.go if present
    Metadata     *SkillMetadata
}

// Check available formats
if info.HasMarkdown() {
    skill, err := info.LoadMarkdown()
}
```

## Unified Loader

`UnifiedLoader` loads skills with format precedence, preferring Go implementations when registered:

```go
ul := loader.NewUnifiedLoader()

// Register Go skill constructors
ul.RegisterGo("weather", func() skill.Skill {
    return &WeatherSkill{APIKey: os.Getenv("WEATHER_API_KEY")}
})

// Load a skill - uses Go if registered, falls back to SKILL.md
skill, format, err := ul.Load("./skills/weather")
fmt.Printf("Loaded %s as %s\n", skill.Name(), format)

// Load all skills from a directory
skills, err := ul.LoadAll("./skills")
```

### Format Precedence

1. **Registered Go constructor** - If `RegisterGo` was called with a matching name
2. **SKILL.md** - Falls back to markdown definition

This allows gradual migration from SKILL.md to native Go without changing calling code.

## Go Skill Registry

For applications that only use Go skills:

```go
registry := loader.NewGoSkillRegistry()

// Register constructors
registry.Register("weather", NewWeatherSkill)
registry.Register("calculator", NewCalculatorSkill)

// Check and retrieve
if registry.Has("weather") {
    skill, err := registry.Get("weather")
}

// List all registered names
names := registry.List()
```

## Type Definitions

### SkillMetadata

```go
type SkillMetadata struct {
    Name        string           `yaml:"name"`
    Description string           `yaml:"description"`
    Metadata    ExtendedMetadata `yaml:"metadata"`
}

type ExtendedMetadata struct {
    OpenClaw *OpenClawMetadata `yaml:"openclaw"`
}

type OpenClawMetadata struct {
    Homepage string        `yaml:"homepage"`
    Requires *Requirements `yaml:"requires"`
    Install  []InstallStep `yaml:"install"`
}

type Requirements struct {
    Bins []string `yaml:"bins"`
}

type InstallStep struct {
    Kind   string   `yaml:"kind"`
    Module string   `yaml:"module"`
    Bins   []string `yaml:"bins"`
    Script string   `yaml:"script"`
}
```

## Best Practices

### 1. Organize Skills by Directory

```
skills/
├── weather/
│   ├── SKILL.md      # Markdown definition
│   └── skill.go      # Optional Go implementation
├── calculator/
│   └── SKILL.md
└── github/
    └── SKILL.md
```

### 2. Verify Dependencies on Init

`MarkdownSkill.Init()` automatically verifies required binaries are available:

```go
skill, _ := loader.LoadMarkdownSkill("SKILL.md")

if err := skill.Init(ctx); err != nil {
    // Required binary not found
    log.Fatal(err)
}
```

### 3. Use UnifiedLoader for Hybrid Deployments

```go
ul := loader.NewUnifiedLoader()

// Register Go implementations for performance-critical skills
ul.RegisterGo("calculator", NewCalculatorSkill)

// Load all - uses Go where available, SKILL.md otherwise
skills, _ := ul.LoadAll("./skills")
```

## See Also

- [Installer](installer.md) - Installing skill dependencies
- [Tools](tools.md) - Tool interface and CommandTool
- [Skills](skills.md) - Skill interface and lifecycle
