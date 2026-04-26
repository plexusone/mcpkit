// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/plexusone/omniskill/skill"
)

// SessionSkill wraps an MCP session as a skill.Skill.
//
// Tools are discovered from the MCP server and cached.
// Tool calls are proxied to the underlying MCP session.
type SessionSkill struct {
	session     *Session
	name        string
	description string
	tools       []skill.Tool
	mu          sync.RWMutex
	initialized bool
}

// AsSkill creates a skill.Skill from this session.
//
// The skill name defaults to "mcp" but can be overridden with options.
// Tools are discovered lazily on first access or when Init() is called.
//
// Example:
//
//	skill := session.AsSkill(
//	    client.WithSkillName("github"),
//	    client.WithSkillDescription("GitHub operations"),
//	)
func (s *Session) AsSkill(opts ...SkillOption) *SessionSkill {
	sk := &SessionSkill{
		session:     s,
		name:        "mcp",
		description: "MCP remote skill",
	}
	for _, opt := range opts {
		opt(sk)
	}
	return sk
}

// SkillOption configures a SessionSkill.
type SkillOption func(*SessionSkill)

// WithSkillName sets the skill name.
func WithSkillName(name string) SkillOption {
	return func(s *SessionSkill) {
		s.name = name
	}
}

// WithSkillDescription sets the skill description.
func WithSkillDescription(desc string) SkillOption {
	return func(s *SessionSkill) {
		s.description = desc
	}
}

// Name returns the skill name.
func (s *SessionSkill) Name() string {
	return s.name
}

// Description returns the skill description.
func (s *SessionSkill) Description() string {
	return s.description
}

// Tools returns all tools from the MCP session.
//
// If Init() has not been called, this will call Init() with
// a background context to discover tools.
func (s *SessionSkill) Tools() []skill.Tool {
	s.mu.RLock()
	if s.initialized {
		tools := s.tools
		s.mu.RUnlock()
		return tools
	}
	s.mu.RUnlock()

	// Initialize with background context
	_ = s.Init(context.Background())

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tools
}

// Init discovers tools from the MCP session.
//
// This fetches the tool list from the server and caches them.
// Subsequent calls to Tools() will return the cached list.
func (s *SessionSkill) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	mcpTools, err := s.session.ListTools(ctx)
	if err != nil {
		return err
	}

	s.tools = make([]skill.Tool, len(mcpTools))
	for i, t := range mcpTools {
		s.tools[i] = &mcpToolWrapper{
			session: s.session,
			tool:    t,
		}
	}
	s.initialized = true

	return nil
}

// Close closes the underlying MCP session.
func (s *SessionSkill) Close() error {
	return s.session.Close()
}

// mcpToolWrapper wraps an mcp.Tool as a skill.Tool.
type mcpToolWrapper struct {
	session *Session
	tool    *mcp.Tool
}

func (t *mcpToolWrapper) Name() string {
	return t.tool.Name
}

func (t *mcpToolWrapper) Description() string {
	return t.tool.Description
}

// Parameters converts the MCP tool's input schema to skill.Parameter map.
func (t *mcpToolWrapper) Parameters() map[string]skill.Parameter {
	if t.tool.InputSchema == nil {
		return nil
	}

	// InputSchema is any - marshal and unmarshal to get consistent map
	schemaBytes, err := json.Marshal(t.tool.InputSchema)
	if err != nil {
		return nil
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}

	// Get required fields
	var required []string
	if reqAny, ok := schema["required"].([]any); ok {
		for _, r := range reqAny {
			if s, ok := r.(string); ok {
				required = append(required, s)
			}
		}
	}
	requiredSet := make(map[string]bool)
	for _, r := range required {
		requiredSet[r] = true
	}

	params := make(map[string]skill.Parameter)
	for name, propAny := range props {
		prop, ok := propAny.(map[string]any)
		if !ok {
			continue
		}
		params[name] = schemaToParameter(prop, requiredSet[name])
	}

	return params
}

// schemaToParameter converts a JSON Schema property to skill.Parameter.
func schemaToParameter(schema map[string]any, required bool) skill.Parameter {
	p := skill.Parameter{
		Required: required,
	}

	if t, ok := schema["type"].(string); ok {
		p.Type = t
	}
	if d, ok := schema["description"].(string); ok {
		p.Description = d
	}
	if def, ok := schema["default"]; ok {
		p.Default = def
	}
	if enum, ok := schema["enum"].([]any); ok {
		p.Enum = enum
	}

	// Handle array items
	if items, ok := schema["items"].(map[string]any); ok {
		itemParam := schemaToParameter(items, false)
		p.Items = &itemParam
	}

	// Handle object properties
	if props, ok := schema["properties"].(map[string]any); ok {
		// Get nested required
		var nestedRequired []string
		if reqAny, ok := schema["required"].([]any); ok {
			for _, r := range reqAny {
				if s, ok := r.(string); ok {
					nestedRequired = append(nestedRequired, s)
				}
			}
		}
		nestedRequiredSet := make(map[string]bool)
		for _, r := range nestedRequired {
			nestedRequiredSet[r] = true
		}

		p.Properties = make(map[string]skill.Parameter)
		for name, propAny := range props {
			if propMap, ok := propAny.(map[string]any); ok {
				p.Properties[name] = schemaToParameter(propMap, nestedRequiredSet[name])
			}
		}
	}

	return p
}

// Call invokes the tool via the MCP session.
func (t *mcpToolWrapper) Call(ctx context.Context, params map[string]any) (any, error) {
	result, err := t.session.CallTool(ctx, t.tool.Name, params)
	if err != nil {
		return nil, err
	}

	// Return structured content if available
	if result.StructuredContent != nil {
		return result.StructuredContent, nil
	}

	// Otherwise return text content
	if len(result.Content) > 0 {
		if tc, ok := result.Content[0].(*mcp.TextContent); ok {
			return tc.Text, nil
		}
	}

	return nil, nil
}

// Ensure SessionSkill implements skill.Skill
var _ skill.Skill = (*SessionSkill)(nil)

// Ensure mcpToolWrapper implements skill.Tool
var _ skill.Tool = (*mcpToolWrapper)(nil)
