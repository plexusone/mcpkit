// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package registry provides skill registration and discovery.
//
// The registry is a central place to register skills and discover
// available tools across all registered skills.
package registry

import (
	"context"
	"errors"
	"sync"

	"github.com/plexusone/omniskill/skill"
)

// ErrSkillNotFound is returned when a skill is not found in the registry.
var ErrSkillNotFound = errors.New("skill not found")

// ErrSkillExists is returned when attempting to register a skill that already exists.
var ErrSkillExists = errors.New("skill already registered")

// Registry manages skill registration and discovery.
type Registry interface {
	// Register adds a skill to the registry.
	// Returns ErrSkillExists if a skill with the same name is already registered.
	Register(s skill.Skill) error

	// Unregister removes a skill from the registry.
	// Returns ErrSkillNotFound if the skill is not registered.
	Unregister(name string) error

	// Get returns a skill by name.
	// Returns ErrSkillNotFound if the skill is not registered.
	Get(name string) (skill.Skill, error)

	// List returns all registered skills.
	List() []skill.Skill

	// ListTools returns all tools across all registered skills.
	// Tool names are prefixed with skill name: "skillname.toolname"
	ListTools() []skill.Tool

	// GetTool returns a specific tool by its full name (skillname.toolname).
	GetTool(fullName string) (skill.Tool, error)

	// Init initializes all registered skills.
	Init(ctx context.Context) error

	// Close closes all registered skills.
	Close() error
}

// InMemory is a thread-safe in-memory registry implementation.
type InMemory struct {
	skills map[string]skill.Skill
	mu     sync.RWMutex
}

// New creates a new in-memory registry.
func New() *InMemory {
	return &InMemory{
		skills: make(map[string]skill.Skill),
	}
}

// Register adds a skill to the registry.
func (r *InMemory) Register(s skill.Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Name()
	if _, exists := r.skills[name]; exists {
		return ErrSkillExists
	}

	r.skills[name] = s
	return nil
}

// Unregister removes a skill from the registry.
func (r *InMemory) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; !exists {
		return ErrSkillNotFound
	}

	delete(r.skills, name)
	return nil
}

// Get returns a skill by name.
func (r *InMemory) Get(name string) (skill.Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, exists := r.skills[name]
	if !exists {
		return nil, ErrSkillNotFound
	}

	return s, nil
}

// List returns all registered skills.
func (r *InMemory) List() []skill.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]skill.Skill, 0, len(r.skills))
	for _, s := range r.skills {
		skills = append(skills, s)
	}
	return skills
}

// ListTools returns all tools across all registered skills.
func (r *InMemory) ListTools() []skill.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []skill.Tool
	for _, s := range r.skills {
		tools = append(tools, s.Tools()...)
	}
	return tools
}

// GetTool returns a specific tool by its full name (skillname.toolname).
func (r *InMemory) GetTool(fullName string) (skill.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Parse skill.tool format
	skillName, toolName := parseToolName(fullName)
	if skillName == "" {
		// No prefix, search all skills
		for _, s := range r.skills {
			for _, t := range s.Tools() {
				if t.Name() == fullName {
					return t, nil
				}
			}
		}
		return nil, errors.New("tool not found: " + fullName)
	}

	// Look up specific skill
	s, exists := r.skills[skillName]
	if !exists {
		return nil, ErrSkillNotFound
	}

	for _, t := range s.Tools() {
		if t.Name() == toolName {
			return t, nil
		}
	}

	return nil, errors.New("tool not found: " + fullName)
}

// Init initializes all registered skills.
func (r *InMemory) Init(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.skills {
		if err := s.Init(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all registered skills.
func (r *InMemory) Close() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errs []error
	for _, s := range r.skills {
		if err := s.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Count returns the number of registered skills.
func (r *InMemory) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// parseToolName splits "skill.tool" into skill and tool names.
// Returns empty skillName if no dot is present.
func parseToolName(fullName string) (skillName, toolName string) {
	for i, c := range fullName {
		if c == '.' {
			return fullName[:i], fullName[i+1:]
		}
	}
	return "", fullName
}

// Ensure InMemory implements Registry.
var _ Registry = (*InMemory)(nil)
