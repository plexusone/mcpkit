// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plexusone/omniskill/skill"
)

// Standard file names for skill definitions.
const (
	// SkillMDFile is the standard name for markdown skill definitions.
	SkillMDFile = "SKILL.md"

	// SkillGoFile is the standard name for Go skill implementations.
	SkillGoFile = "skill.go"
)

// SkillFormat represents the format of a skill definition.
type SkillFormat string

const (
	// FormatMarkdown is an OpenClaw SKILL.md definition.
	FormatMarkdown SkillFormat = "markdown"

	// FormatGo is a native Go skill implementation.
	FormatGo SkillFormat = "go"
)

// SkillInfo contains metadata about a skill found in a directory.
type SkillInfo struct {
	// Dir is the skill directory path.
	Dir string

	// Name is the skill name (from directory name or SKILL.md).
	Name string

	// Formats lists the available formats for this skill.
	Formats []SkillFormat

	// MarkdownPath is the path to SKILL.md if present.
	MarkdownPath string

	// GoPath is the path to skill.go if present.
	GoPath string

	// Metadata is parsed from SKILL.md frontmatter if available.
	Metadata *SkillMetadata
}

// HasMarkdown returns true if a SKILL.md file exists.
func (i *SkillInfo) HasMarkdown() bool {
	return i.MarkdownPath != ""
}

// HasGo returns true if a skill.go file exists.
func (i *SkillInfo) HasGo() bool {
	return i.GoPath != ""
}

// Inspect examines a directory and returns information about available skill formats.
func Inspect(dir string) (*SkillInfo, error) {
	info := &SkillInfo{
		Dir:  dir,
		Name: filepath.Base(dir),
	}

	// Check for SKILL.md
	mdPath := filepath.Join(dir, SkillMDFile)
	if _, err := os.Stat(mdPath); err == nil {
		info.MarkdownPath = mdPath
		info.Formats = append(info.Formats, FormatMarkdown)

		// Parse metadata from SKILL.md
		content, err := os.ReadFile(mdPath)
		if err == nil {
			metadata, _, _ := parseFrontmatter(string(content))
			info.Metadata = &metadata
			if metadata.Name != "" {
				info.Name = metadata.Name
			}
		}
	}

	// Check for skill.go
	goPath := filepath.Join(dir, SkillGoFile)
	if _, err := os.Stat(goPath); err == nil {
		info.GoPath = goPath
		info.Formats = append(info.Formats, FormatGo)
	}

	if len(info.Formats) == 0 {
		return nil, fmt.Errorf("no skill definition found in %s", dir)
	}

	return info, nil
}

// DiscoverSkills finds all skill directories under a root path.
func DiscoverSkills(root string) ([]*SkillInfo, error) {
	var skills []*SkillInfo

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(root, entry.Name())
		info, err := Inspect(dir)
		if err != nil {
			// Skip directories without skill definitions
			continue
		}

		skills = append(skills, info)
	}

	return skills, nil
}

// LoadMarkdown loads the SKILL.md from a skill directory.
// Returns an error if no SKILL.md exists.
func (i *SkillInfo) LoadMarkdown() (*MarkdownSkill, error) {
	if !i.HasMarkdown() {
		return nil, fmt.Errorf("no SKILL.md in %s", i.Dir)
	}
	return LoadMarkdownSkill(i.MarkdownPath)
}

// GoSkillRegistry allows registration of Go skill constructors by name.
// This enables loading Go skills dynamically by directory name.
type GoSkillRegistry struct {
	constructors map[string]GoSkillConstructor
}

// GoSkillConstructor is a function that creates a Go skill instance.
type GoSkillConstructor func() skill.Skill

// NewGoSkillRegistry creates a new Go skill registry.
func NewGoSkillRegistry() *GoSkillRegistry {
	return &GoSkillRegistry{
		constructors: make(map[string]GoSkillConstructor),
	}
}

// Register adds a Go skill constructor for a given name.
func (r *GoSkillRegistry) Register(name string, constructor GoSkillConstructor) {
	r.constructors[name] = constructor
}

// Get returns a new instance of the Go skill with the given name.
func (r *GoSkillRegistry) Get(name string) (skill.Skill, error) {
	constructor, ok := r.constructors[name]
	if !ok {
		return nil, fmt.Errorf("no Go skill registered for %q", name)
	}
	return constructor(), nil
}

// Has returns true if a Go skill is registered with the given name.
func (r *GoSkillRegistry) Has(name string) bool {
	_, ok := r.constructors[name]
	return ok
}

// List returns all registered Go skill names.
func (r *GoSkillRegistry) List() []string {
	names := make([]string, 0, len(r.constructors))
	for name := range r.constructors {
		names = append(names, name)
	}
	return names
}

// UnifiedLoader loads skills from directories, preferring Go implementations
// when available but falling back to SKILL.md.
type UnifiedLoader struct {
	goRegistry *GoSkillRegistry
}

// NewUnifiedLoader creates a new unified loader.
func NewUnifiedLoader() *UnifiedLoader {
	return &UnifiedLoader{
		goRegistry: NewGoSkillRegistry(),
	}
}

// RegisterGo registers a Go skill constructor.
func (l *UnifiedLoader) RegisterGo(name string, constructor GoSkillConstructor) {
	l.goRegistry.Register(name, constructor)
}

// Load loads a skill from a directory, preferring Go over SKILL.md.
func (l *UnifiedLoader) Load(dir string) (skill.Skill, SkillFormat, error) {
	info, err := Inspect(dir)
	if err != nil {
		return nil, "", err
	}

	return l.LoadInfo(info)
}

// LoadInfo loads a skill from SkillInfo, preferring Go over SKILL.md.
func (l *UnifiedLoader) LoadInfo(info *SkillInfo) (skill.Skill, SkillFormat, error) {
	// Prefer Go implementation if registered
	if info.HasGo() && l.goRegistry.Has(info.Name) {
		s, err := l.goRegistry.Get(info.Name)
		if err != nil {
			return nil, "", err
		}
		return s, FormatGo, nil
	}

	// Fall back to SKILL.md
	if info.HasMarkdown() {
		s, err := info.LoadMarkdown()
		if err != nil {
			return nil, "", err
		}
		return s, FormatMarkdown, nil
	}

	return nil, "", fmt.Errorf("no loadable skill format in %s", info.Dir)
}

// LoadAll loads all skills from a directory.
func (l *UnifiedLoader) LoadAll(root string) ([]skill.Skill, error) {
	infos, err := DiscoverSkills(root)
	if err != nil {
		return nil, err
	}

	var skills []skill.Skill
	for _, info := range infos {
		s, _, err := l.LoadInfo(info)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", info.Name, err)
		}
		skills = append(skills, s)
	}

	return skills, nil
}
