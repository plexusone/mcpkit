# Installer

The `installer` package handles installation and management of skill dependencies. It supports multiple package managers and provides a unified interface for dependency resolution.

## Manager

`Manager` handles installation of dependencies defined in SKILL.md files:

```go
import "github.com/plexusone/omniskill/installer"

mgr := installer.NewManager()

// Install dependencies
steps := []loader.InstallStep{
    {Kind: "go", Module: "github.com/user/tool@latest", Bins: []string{"tool"}},
    {Kind: "npm", Module: "some-cli"},
}

if err := mgr.Install(ctx, steps); err != nil {
    log.Fatal(err)
}
```

### Supported Package Managers

| Kind | Command | Example |
|------|---------|---------|
| `go` | `go install` | `github.com/user/tool@latest` |
| `npm` | `npm install -g` | `some-cli@1.0.0` |
| `pip` | `pip install` | `some-package` |
| `docker` | `docker pull` | `alpine:latest` |
| `brew` | `brew install` | `jq` |

### Manager Options

```go
mgr := installer.NewManager()

// Set installation timeout (default: 5 minutes)
mgr.Timeout = 10 * time.Minute

// Add environment variables
mgr.Env = []string{"GOBIN=/custom/bin"}

// Enable verbose output
mgr.Verbose = true

// Dry-run mode (only report what would be installed)
mgr.DryRun = true
```

## Verifying Binaries

Check if required binaries are available:

```go
bins := []string{"notcrawl", "jq", "gh"}

missing := mgr.VerifyBinaries(bins)
if len(missing) > 0 {
    fmt.Printf("Missing binaries: %v\n", missing)
}
```

## Installing Missing Dependencies

Only install dependencies whose binaries are not found:

```go
// Load skill and get install steps
skill, _ := loader.LoadMarkdownSkill("SKILL.md")
steps := skill.GetInstallSteps()

// Install only what's missing
if err := mgr.InstallMissing(ctx, steps); err != nil {
    log.Fatal(err)
}
```

## Installation Results

Get detailed results for each installation step:

```go
results := mgr.InstallWithResults(ctx, steps)

for _, r := range results {
    if r.Success {
        fmt.Printf("✓ %s (%s)\n", r.Step.Module, r.Duration)
    } else {
        fmt.Printf("✗ %s: %v\n", r.Step.Module, r.Error)
    }
}
```

### InstallResult

```go
type InstallResult struct {
    Step     loader.InstallStep
    Success  bool
    Error    error
    Duration time.Duration
}
```

## Custom Installers

Register custom installation handlers:

```go
mgr.RegisterInstaller("cargo", func(ctx context.Context, step loader.InstallStep) error {
    cmd := exec.CommandContext(ctx, "cargo", "install", step.Module)
    return cmd.Run()
})
```

## SkillInstaller

`SkillInstaller` installs complete skills from various sources:

```go
si := installer.NewSkillInstaller()

// Install from git repository
skill, err := si.Install(ctx, "github.com/user/repo@v1.0.0")

// Install from git subdirectory
skill, err := si.Install(ctx, "github.com/user/repo/skills/weather")

// Install from local path (copy)
skill, err := si.Install(ctx, "./local/skill")

// Install from local path (symlink for development)
si.Symlink = true
skill, err := si.Install(ctx, "./local/skill")
```

### Source Formats

| Format | Example | Description |
|--------|---------|-------------|
| Git (GitHub) | `github.com/user/repo` | Clone repository |
| Git with ref | `github.com/user/repo@v1.0.0` | Clone specific tag/branch |
| Git subdirectory | `github.com/user/repo/skills/weather` | Extract subdirectory |
| Git URL | `https://github.com/user/repo.git` | Explicit URL |
| Local path | `./my-skill` | Copy or symlink |
| Absolute path | `/path/to/skill` | Copy or symlink |

### SkillInstaller Options

```go
si := installer.NewSkillInstaller()

// Install to local ./skills directory (default)
si.SkillsDir = "./skills"

// Install to global ~/.omniskill/skills
si.UseGlobal = true
si.GlobalDir = "/custom/global/path"

// Use symlinks for local installs (development mode)
si.Symlink = true

// Verbose output
si.Verbose = true
```

### Installation Directories

| Type | Path | Use Case |
|------|------|----------|
| Local | `./skills/` | Project-specific skills |
| Global | `~/.omniskill/skills/` | Shared across projects |

## Managing Installed Skills

### List Installed Skills

```go
skills, err := si.List()

for _, s := range skills {
    location := "local"
    if s.Global {
        location = "global"
    }

    symlink := ""
    if s.Symlinked {
        symlink = " (symlinked)"
    }

    fmt.Printf("%s [%s]%s\n", s.Name, location, symlink)
}
```

### Uninstall Skills

```go
if err := si.Uninstall("weather"); err != nil {
    log.Fatal(err)
}
```

### InstalledSkill

```go
type InstalledSkill struct {
    Name       string      // Skill directory name
    Path       string      // Full installation path
    Source     *Source     // Original source location
    SourceType SourceType  // git or local
    Global     bool        // Installed globally
    Symlinked  bool        // Is a symlink (local installs)
}
```

## Complete Workflow

Install a skill and its dependencies:

```go
import (
    "github.com/plexusone/omniskill/installer"
    "github.com/plexusone/omniskill/loader"
)

// 1. Install the skill from source
si := installer.NewSkillInstaller()
installed, err := si.Install(ctx, "github.com/user/notcrawl-skill@v1.0.0")
if err != nil {
    log.Fatal(err)
}

// 2. Load the skill
skill, err := loader.LoadMarkdownSkillDir(installed.Path)
if err != nil {
    log.Fatal(err)
}

// 3. Install missing dependencies
mgr := installer.NewManager()
steps := skill.GetInstallSteps()
if err := mgr.InstallMissing(ctx, steps); err != nil {
    log.Fatal(err)
}

// 4. Initialize and use
if err := skill.Init(ctx); err != nil {
    log.Fatal(err)
}

for _, tool := range skill.Tools() {
    fmt.Printf("Ready: %s\n", tool.Name())
}
```

## Development Workflow

For skill development, use symlinks to avoid copying:

```go
si := installer.NewSkillInstaller()
si.Symlink = true

// Symlink local skill for development
_, err := si.Install(ctx, "../my-skills/weather")

// Changes to ../my-skills/weather are immediately reflected
```

## Error Handling

```go
mgr := installer.NewManager()

err := mgr.Install(ctx, steps)
if err != nil {
    // Check for specific failures
    switch {
    case strings.Contains(err.Error(), "unsupported installer kind"):
        // Unknown package manager
    case strings.Contains(err.Error(), "binaries not found after install"):
        // Binary verification failed
    default:
        // Installation command failed
    }
}
```

## See Also

- [Loader](loader.md) - Loading skills and SKILL.md format
- [Skills](skills.md) - Skill interface and lifecycle
- [Tools](tools.md) - Tool interface and CommandTool
