// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package skill defines the core interfaces for skills and tools.
//
// A Skill is a named collection of related tools. Skills can be:
//   - Built natively in Go
//   - Imported from MCP servers
//   - Generated from OpenAPI specifications
//
// Skills are the primary abstraction in omniskill, enabling "define once,
// deploy everywhere" across MCP, compiled Go, and other formats.
package skill

import "context"

// Skill represents a named collection of related tools.
//
// Skills encapsulate a logical grouping of functionality (e.g., "github",
// "weather", "database") and provide lifecycle management via Init/Close.
//
// Example implementation:
//
//	type WeatherSkill struct {
//	    apiKey string
//	    client *http.Client
//	}
//
//	func (s *WeatherSkill) Name() string        { return "weather" }
//	func (s *WeatherSkill) Description() string { return "Weather forecasts and conditions" }
//	func (s *WeatherSkill) Tools() []Tool       { return []Tool{s.getCurrentWeather(), s.getForecast()} }
//	func (s *WeatherSkill) Init(ctx context.Context) error { s.client = &http.Client{}; return nil }
//	func (s *WeatherSkill) Close() error        { return nil }
type Skill interface {
	// Name returns the skill identifier (e.g., "github", "weather").
	// Names should be lowercase, alphanumeric with underscores.
	Name() string

	// Description returns a human-readable description of what the skill does.
	Description() string

	// Tools returns the tools provided by this skill.
	// This may be called multiple times and should return consistent results
	// after Init() completes.
	Tools() []Tool

	// Init initializes the skill before use.
	// Called once when the skill is registered. Use this to establish
	// connections, load configuration, or perform other setup.
	Init(ctx context.Context) error

	// Close releases any resources held by the skill.
	// Called when the skill is unregistered or the application shuts down.
	Close() error
}

// BaseSkill provides a minimal Skill implementation that can be embedded.
// It implements all Skill methods with sensible defaults.
type BaseSkill struct {
	SkillName        string
	SkillDescription string
	SkillTools       []Tool
}

// Name returns the skill name.
func (s *BaseSkill) Name() string {
	return s.SkillName
}

// Description returns the skill description.
func (s *BaseSkill) Description() string {
	return s.SkillDescription
}

// Tools returns the skill's tools.
func (s *BaseSkill) Tools() []Tool {
	return s.SkillTools
}

// Init is a no-op by default.
func (s *BaseSkill) Init(ctx context.Context) error {
	return nil
}

// Close is a no-op by default.
func (s *BaseSkill) Close() error {
	return nil
}

// Ensure BaseSkill implements Skill.
var _ Skill = (*BaseSkill)(nil)
