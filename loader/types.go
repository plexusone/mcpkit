// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package loader provides skill loaders for different formats.
//
// Loaders convert external skill definitions (SKILL.md, OpenAPI, etc.)
// into the standard skill.Skill interface, enabling "define once,
// deploy everywhere" across different skill formats.
package loader

// SkillMetadata contains metadata about a skill definition.
type SkillMetadata struct {
	// Name is the skill identifier.
	Name string `yaml:"name" json:"name"`

	// Description is a human-readable description.
	Description string `yaml:"description" json:"description"`

	// Metadata contains extended metadata.
	Metadata ExtendedMetadata `yaml:"metadata" json:"metadata"`
}

// ExtendedMetadata contains provider-specific metadata.
type ExtendedMetadata struct {
	// OpenClaw contains OpenClaw-specific metadata.
	OpenClaw *OpenClawMetadata `yaml:"openclaw" json:"openclaw,omitempty"`
}

// OpenClawMetadata contains OpenClaw SKILL.md specific metadata.
type OpenClawMetadata struct {
	// Homepage is the project homepage URL.
	Homepage string `yaml:"homepage" json:"homepage,omitempty"`

	// Requires lists the skill's requirements.
	Requires *Requirements `yaml:"requires" json:"requires,omitempty"`

	// Install contains installation instructions.
	Install []InstallStep `yaml:"install" json:"install,omitempty"`
}

// Requirements lists what a skill needs to function.
type Requirements struct {
	// Bins lists required binary executables.
	Bins []string `yaml:"bins" json:"bins,omitempty"`
}

// InstallStep describes how to install a dependency.
type InstallStep struct {
	// Kind is the installation method: "go", "npm", "pip", "docker", etc.
	Kind string `yaml:"kind" json:"kind"`

	// Module is the package/module identifier.
	// For Go: github.com/user/pkg@version
	// For npm: package-name@version
	Module string `yaml:"module" json:"module"`

	// Bins lists the binaries installed by this step.
	Bins []string `yaml:"bins" json:"bins,omitempty"`

	// Script is an optional post-install script.
	Script string `yaml:"script" json:"script,omitempty"`
}

// SkillType identifies the type of skill definition.
type SkillType string

const (
	// SkillTypeNativeGo is a compiled Go skill.
	SkillTypeNativeGo SkillType = "native_go"

	// SkillTypeMarkdown is a SKILL.md markdown skill.
	SkillTypeMarkdown SkillType = "markdown"

	// SkillTypeMCPServer is an MCP server skill.
	SkillTypeMCPServer SkillType = "mcp_server"

	// SkillTypeOpenAPI is an OpenAPI-defined skill.
	SkillTypeOpenAPI SkillType = "openapi"
)

// DiscoveredCommand represents a command found in markdown code blocks.
type DiscoveredCommand struct {
	// Command is the executable name.
	Command string

	// Args are the command arguments.
	Args []string

	// FullLine is the complete command line.
	FullLine string

	// Description is extracted from surrounding context.
	Description string
}
