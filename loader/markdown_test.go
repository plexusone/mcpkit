// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package loader

import (
	"context"
	"testing"
)

func TestParseMarkdownSkill(t *testing.T) {
	content := `---
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
---
# Usage

Search your Notion archive:

` + "```bash" + `
notcrawl search "query"
` + "```" + `

Sync your archive:

` + "```bash" + `
notcrawl sync --all
` + "```" + `
`

	skill, err := ParseMarkdownSkill(content, "test/SKILL.md")
	if err != nil {
		t.Fatalf("ParseMarkdownSkill failed: %v", err)
	}

	// Verify metadata
	if skill.Name() != "notcrawl" {
		t.Errorf("Name = %q, want %q", skill.Name(), "notcrawl")
	}

	if skill.Description() != "Notion archive search and sync" {
		t.Errorf("Description = %q, want %q", skill.Description(), "Notion archive search and sync")
	}

	// Verify OpenClaw metadata
	if skill.Metadata.Metadata.OpenClaw == nil {
		t.Fatal("OpenClaw metadata is nil")
	}

	if skill.Metadata.Metadata.OpenClaw.Homepage != "https://github.com/user/notcrawl" {
		t.Errorf("Homepage = %q, want %q", skill.Metadata.Metadata.OpenClaw.Homepage, "https://github.com/user/notcrawl")
	}

	// Verify requirements
	if skill.Metadata.Metadata.OpenClaw.Requires == nil {
		t.Fatal("Requires is nil")
	}

	if len(skill.Metadata.Metadata.OpenClaw.Requires.Bins) != 1 {
		t.Fatalf("Bins length = %d, want 1", len(skill.Metadata.Metadata.OpenClaw.Requires.Bins))
	}

	if skill.Metadata.Metadata.OpenClaw.Requires.Bins[0] != "notcrawl" {
		t.Errorf("Bins[0] = %q, want %q", skill.Metadata.Metadata.OpenClaw.Requires.Bins[0], "notcrawl")
	}

	// Verify install steps
	if len(skill.Metadata.Metadata.OpenClaw.Install) != 1 {
		t.Fatalf("Install length = %d, want 1", len(skill.Metadata.Metadata.OpenClaw.Install))
	}

	step := skill.Metadata.Metadata.OpenClaw.Install[0]
	if step.Kind != "go" {
		t.Errorf("Install.Kind = %q, want %q", step.Kind, "go")
	}
	if step.Module != "github.com/user/notcrawl@latest" {
		t.Errorf("Install.Module = %q, want %q", step.Module, "github.com/user/notcrawl@latest")
	}

	// Verify guidance
	if skill.GetGuidance() == "" {
		t.Error("Guidance is empty")
	}

	// Verify tools were generated
	tools := skill.Tools()
	if len(tools) == 0 {
		t.Error("No tools generated")
	}

	// Should have a "run" tool for the notcrawl binary
	var hasRunTool bool
	for _, tool := range tools {
		if tool.Name() == "run" {
			hasRunTool = true
			break
		}
	}
	if !hasRunTool {
		t.Error("Expected 'run' tool not found")
	}
}

func TestParseMarkdownSkill_MissingFrontmatter(t *testing.T) {
	content := `# No frontmatter here

Just some markdown content.
`

	_, err := ParseMarkdownSkill(content, "test/SKILL.md")
	if err == nil {
		t.Error("Expected error for missing frontmatter")
	}
}

func TestParseMarkdownSkill_UnclosedFrontmatter(t *testing.T) {
	content := `---
name: test
description: "Test skill"
# Missing closing ---
`

	_, err := ParseMarkdownSkill(content, "test/SKILL.md")
	if err == nil {
		t.Error("Expected error for unclosed frontmatter")
	}
}

func TestDiscoverCommands(t *testing.T) {
	markdown := `
Search for items:

` + "```bash" + `
gh search issues "query"
` + "```" + `

Clone a repo:

` + "```bash" + `
git clone <repo-url>
` + "```" + `
`

	commands := discoverCommands(markdown, &OpenClawMetadata{
		Requires: &Requirements{
			Bins: []string{"gh", "git"},
		},
	})

	if len(commands) != 2 {
		t.Fatalf("Commands length = %d, want 2", len(commands))
	}

	// Verify first command
	if commands[0].Command != "gh" {
		t.Errorf("Command[0].Command = %q, want %q", commands[0].Command, "gh")
	}

	// Verify second command
	if commands[1].Command != "git" {
		t.Errorf("Command[1].Command = %q, want %q", commands[1].Command, "git")
	}
}

func TestExtractParameters(t *testing.T) {
	cmd := DiscoveredCommand{
		Command: "notcrawl",
		Args:    []string{"search", "<query>", "--format", "<format>"},
	}

	params := extractParameters(cmd)

	if _, ok := params["query"]; !ok {
		t.Error("Expected 'query' parameter not found")
	}

	if _, ok := params["format"]; !ok {
		t.Error("Expected 'format' parameter not found")
	}
}

func TestMarkdownSkill_Init(t *testing.T) {
	// Create a skill that doesn't require any binaries
	skill := &MarkdownSkill{
		Metadata: SkillMetadata{
			Name:        "test",
			Description: "Test skill",
		},
	}

	// Init should succeed
	if err := skill.Init(context.Background()); err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Close should succeed
	if err := skill.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestGenerateToolName(t *testing.T) {
	tests := []struct {
		cmd  DiscoveredCommand
		want string
	}{
		{
			cmd:  DiscoveredCommand{Command: "gh", Args: []string{"search", "issues"}},
			want: "gh_search",
		},
		{
			cmd:  DiscoveredCommand{Command: "git", Args: []string{"clone", "<url>"}},
			want: "git_clone",
		},
		{
			cmd:  DiscoveredCommand{Command: "npm", Args: []string{"--version"}},
			want: "",
		},
		{
			cmd:  DiscoveredCommand{Command: "test", Args: []string{}},
			want: "",
		},
	}

	for _, tt := range tests {
		got := generateToolName(tt.cmd)
		if got != tt.want {
			t.Errorf("generateToolName(%v) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}
