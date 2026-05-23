// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package loader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/plexusone/omniskill/skill"
)

// MarkdownSkill implements skill.Skill for SKILL.md definitions.
//
// MarkdownSkill parses OpenClaw-format SKILL.md files and exposes them
// as standard skills. The markdown content serves as guidance for AI
// agents, while discovered commands become callable Tools.
//
// Example SKILL.md format:
//
//	---
//	name: notcrawl
//	description: "Notion archive search and sync"
//	metadata:
//	  openclaw:
//	    requires:
//	      bins: [notcrawl]
//	    install:
//	      - kind: go
//	        module: github.com/user/notcrawl@latest
//	---
//	# Usage
//	```bash
//	notcrawl search "query"
//	```
type MarkdownSkill struct {
	// Metadata from YAML frontmatter.
	Metadata SkillMetadata

	// Guidance is the full markdown body (for AI context).
	Guidance string

	// Commands are discovered from code blocks.
	Commands []DiscoveredCommand

	// SourcePath is the SKILL.md file path.
	SourcePath string

	// tools are generated from commands.
	tools []skill.Tool
}

// LoadMarkdownSkill loads a skill from a SKILL.md file.
func LoadMarkdownSkill(path string) (*MarkdownSkill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill file: %w", err)
	}

	return ParseMarkdownSkill(string(content), path)
}

// LoadMarkdownSkillDir loads a skill from a directory containing SKILL.md.
func LoadMarkdownSkillDir(dir string) (*MarkdownSkill, error) {
	skillPath := filepath.Join(dir, "SKILL.md")
	return LoadMarkdownSkill(skillPath)
}

// ParseMarkdownSkill parses a SKILL.md content string.
func ParseMarkdownSkill(content, sourcePath string) (*MarkdownSkill, error) {
	// Split frontmatter and body
	metadata, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	// Discover commands from code blocks
	commands := discoverCommands(body, metadata.Metadata.OpenClaw)

	ms := &MarkdownSkill{
		Metadata:   metadata,
		Guidance:   body,
		Commands:   commands,
		SourcePath: sourcePath,
	}

	// Generate tools from commands
	ms.tools = ms.generateTools()

	return ms, nil
}

// Name returns the skill name.
func (s *MarkdownSkill) Name() string {
	return s.Metadata.Name
}

// Description returns the skill description.
func (s *MarkdownSkill) Description() string {
	return s.Metadata.Description
}

// Tools returns the tools discovered from the skill.
func (s *MarkdownSkill) Tools() []skill.Tool {
	return s.tools
}

// Init initializes the skill.
func (s *MarkdownSkill) Init(ctx context.Context) error {
	// Verify required binaries are available
	if s.Metadata.Metadata.OpenClaw != nil && s.Metadata.Metadata.OpenClaw.Requires != nil {
		for _, bin := range s.Metadata.Metadata.OpenClaw.Requires.Bins {
			if _, err := lookPath(bin); err != nil {
				return fmt.Errorf("required binary %q not found: %w", bin, err)
			}
		}
	}
	return nil
}

// Close releases skill resources.
func (s *MarkdownSkill) Close() error {
	return nil
}

// GetGuidance returns the full markdown guidance for AI context.
func (s *MarkdownSkill) GetGuidance() string {
	return s.Guidance
}

// GetInstallSteps returns the installation instructions.
func (s *MarkdownSkill) GetInstallSteps() []InstallStep {
	if s.Metadata.Metadata.OpenClaw != nil {
		return s.Metadata.Metadata.OpenClaw.Install
	}
	return nil
}

// generateTools creates Tool implementations from discovered commands.
func (s *MarkdownSkill) generateTools() []skill.Tool {
	// Group commands by executable
	cmdGroups := make(map[string][]DiscoveredCommand)
	for _, cmd := range s.Commands {
		cmdGroups[cmd.Command] = append(cmdGroups[cmd.Command], cmd)
	}

	var tools []skill.Tool

	// Create a general "run" tool for the primary command
	if s.Metadata.Metadata.OpenClaw != nil && s.Metadata.Metadata.OpenClaw.Requires != nil {
		for _, bin := range s.Metadata.Metadata.OpenClaw.Requires.Bins {
			tools = append(tools, &skill.CommandTool{
				ToolName:        "run",
				ToolDescription: fmt.Sprintf("Run %s command with arguments", bin),
				Command:         bin,
				Args:            []string{"{{args}}"},
				ToolParameters: map[string]skill.Parameter{
					"args": {
						Type:        "string",
						Description: "Command arguments (space-separated)",
						Required:    true,
					},
				},
			})
		}
	}

	// Create specific tools for common command patterns
	for _, cmd := range s.Commands {
		if len(cmd.Args) > 0 {
			toolName := generateToolName(cmd)
			if toolName == "" {
				continue
			}

			// Check if we already have this tool
			exists := false
			for _, t := range tools {
				if t.Name() == toolName {
					exists = true
					break
				}
			}
			if exists {
				continue
			}

			params := extractParameters(cmd)
			tools = append(tools, &skill.CommandTool{
				ToolName:        toolName,
				ToolDescription: cmd.Description,
				Command:         cmd.Command,
				Args:            buildArgTemplate(cmd),
				ToolParameters:  params,
			})
		}
	}

	return tools
}

// parseFrontmatter extracts YAML frontmatter and markdown body.
func parseFrontmatter(content string) (SkillMetadata, string, error) {
	var metadata SkillMetadata

	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return metadata, content, fmt.Errorf("missing YAML frontmatter")
	}

	// Find closing ---
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return metadata, content, fmt.Errorf("unclosed YAML frontmatter")
	}

	// Parse YAML
	yamlContent := strings.Join(lines[1:endIdx], "\n")
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		return metadata, "", fmt.Errorf("parse YAML: %w", err)
	}

	// Extract body
	body := strings.Join(lines[endIdx+1:], "\n")
	body = strings.TrimSpace(body)

	return metadata, body, nil
}

// discoverCommands finds commands in markdown code blocks.
func discoverCommands(markdown string, ocMeta *OpenClawMetadata) []DiscoveredCommand {
	var commands []DiscoveredCommand
	var allowedCommands []string

	// Build list of allowed commands from requirements
	if ocMeta != nil && ocMeta.Requires != nil {
		allowedCommands = ocMeta.Requires.Bins
	}

	// Also add common shell commands
	commonCommands := []string{"gh", "git", "npm", "docker", "curl"}
	allowedCommands = append(allowedCommands, commonCommands...)

	// Find code blocks
	scanner := bufio.NewScanner(strings.NewReader(markdown))
	inCodeBlock := false
	var currentBlock []string
	var lastDescription string

	for scanner.Scan() {
		line := scanner.Text()

		// Track descriptions (lines before code blocks)
		if !inCodeBlock && !strings.HasPrefix(line, "```") && strings.TrimSpace(line) != "" {
			// Keep track of potential description
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "-") {
				lastDescription = trimmed
			} else if strings.HasPrefix(trimmed, "-") {
				lastDescription = strings.TrimPrefix(trimmed, "- ")
			}
		}

		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End of code block - process commands
				for _, cmdLine := range currentBlock {
					cmd := parseCommandLine(cmdLine, allowedCommands)
					if cmd != nil {
						cmd.Description = lastDescription
						commands = append(commands, *cmd)
					}
				}
				currentBlock = nil
				lastDescription = ""
			}
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				currentBlock = append(currentBlock, trimmed)
			}
		}
	}

	return commands
}

// parseCommandLine parses a command line into a DiscoveredCommand.
func parseCommandLine(line string, allowedCommands []string) *DiscoveredCommand {
	// Remove common prefixes
	line = strings.TrimPrefix(line, "$ ")

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]

	// Check if this is an allowed command
	allowed := false
	for _, ac := range allowedCommands {
		if cmd == ac {
			allowed = true
			break
		}
	}

	if !allowed {
		return nil
	}

	return &DiscoveredCommand{
		Command:  cmd,
		Args:     parts[1:],
		FullLine: line,
	}
}

// generateToolName creates a tool name from a command.
func generateToolName(cmd DiscoveredCommand) string {
	if len(cmd.Args) == 0 {
		return ""
	}

	// Use first non-flag argument as subcommand
	var subcommand string
	for _, arg := range cmd.Args {
		if !strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "<") && !strings.HasPrefix(arg, "\"") {
			subcommand = arg
			break
		}
	}

	if subcommand == "" {
		return ""
	}

	// Create tool name: command_subcommand
	name := cmd.Command + "_" + subcommand
	name = strings.ReplaceAll(name, "-", "_")

	return name
}

// extractParameters extracts parameter definitions from a command.
func extractParameters(cmd DiscoveredCommand) map[string]skill.Parameter {
	params := make(map[string]skill.Parameter)

	// Look for <param> and "query" patterns
	paramRegex := regexp.MustCompile(`<([^>]+)>`)
	quotedRegex := regexp.MustCompile(`"([^"]+)"`)

	for _, arg := range cmd.Args {
		// Handle <param> patterns
		if matches := paramRegex.FindStringSubmatch(arg); len(matches) > 1 {
			paramName := strings.ReplaceAll(matches[1], "-", "_")
			params[paramName] = skill.Parameter{
				Type:        "string",
				Description: fmt.Sprintf("The %s parameter", matches[1]),
				Required:    true,
			}
		}

		// Handle "query" patterns (common placeholder)
		if matches := quotedRegex.FindStringSubmatch(arg); len(matches) > 1 {
			if matches[1] == "query" || matches[1] == "search" {
				params["query"] = skill.Parameter{
					Type:        "string",
					Description: "Search query",
					Required:    true,
				}
			}
		}
	}

	// If no params found, add a generic args parameter
	if len(params) == 0 {
		params["args"] = skill.Parameter{
			Type:        "string",
			Description: "Additional arguments",
			Required:    false,
		}
	}

	return params
}

// buildArgTemplate creates argument template from command.
func buildArgTemplate(cmd DiscoveredCommand) []string {
	var args []string

	paramRegex := regexp.MustCompile(`<([^>]+)>`)
	quotedRegex := regexp.MustCompile(`"[^"]+"`)

	for _, arg := range cmd.Args {
		// Replace <param> with {{param}}
		if matches := paramRegex.FindStringSubmatch(arg); len(matches) > 1 {
			paramName := strings.ReplaceAll(matches[1], "-", "_")
			args = append(args, "{{"+paramName+"}}")
			continue
		}

		// Replace quoted strings with {{query}} or similar
		if quotedRegex.MatchString(arg) {
			args = append(args, "{{query}}")
			continue
		}

		// Keep literal args
		args = append(args, arg)
	}

	return args
}

// lookPath is a wrapper for exec.LookPath for testing.
var lookPath = func(file string) (string, error) {
	return exec.LookPath(file)
}

// Ensure MarkdownSkill implements skill.Skill.
var _ skill.Skill = (*MarkdownSkill)(nil)
