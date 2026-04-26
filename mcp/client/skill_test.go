// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package client

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/plexusone/omniskill/skill"
)

func TestSessionSkill_Interface(t *testing.T) {
	// Verify SessionSkill implements skill.Skill
	var _ skill.Skill = (*SessionSkill)(nil)
}

func TestMcpToolWrapper_Interface(t *testing.T) {
	// Verify mcpToolWrapper implements skill.Tool
	var _ skill.Tool = (*mcpToolWrapper)(nil)
}

func TestSchemaToParameter(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]any
		required bool
		check    func(t *testing.T, p skill.Parameter)
	}{
		{
			name: "basic string",
			schema: map[string]any{
				"type":        "string",
				"description": "A string parameter",
			},
			required: true,
			check: func(t *testing.T, p skill.Parameter) {
				if p.Type != "string" {
					t.Errorf("expected type 'string', got %q", p.Type)
				}
				if p.Description != "A string parameter" {
					t.Errorf("expected description 'A string parameter', got %q", p.Description)
				}
				if !p.Required {
					t.Error("expected required=true")
				}
			},
		},
		{
			name: "integer with default",
			schema: map[string]any{
				"type":    "integer",
				"default": 42,
			},
			required: false,
			check: func(t *testing.T, p skill.Parameter) {
				if p.Type != "integer" {
					t.Errorf("expected type 'integer', got %q", p.Type)
				}
				if p.Default != 42 {
					t.Errorf("expected default 42, got %v", p.Default)
				}
				if p.Required {
					t.Error("expected required=false")
				}
			},
		},
		{
			name: "string with enum",
			schema: map[string]any{
				"type": "string",
				"enum": []any{"a", "b", "c"},
			},
			required: false,
			check: func(t *testing.T, p skill.Parameter) {
				if len(p.Enum) != 3 {
					t.Errorf("expected 3 enum values, got %d", len(p.Enum))
				}
			},
		},
		{
			name: "array with items",
			schema: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
			required: false,
			check: func(t *testing.T, p skill.Parameter) {
				if p.Type != "array" {
					t.Errorf("expected type 'array', got %q", p.Type)
				}
				if p.Items == nil {
					t.Fatal("expected items to be set")
				}
				if p.Items.Type != "string" {
					t.Errorf("expected items type 'string', got %q", p.Items.Type)
				}
			},
		},
		{
			name: "object with properties",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"x": map[string]any{"type": "number"},
					"y": map[string]any{"type": "number"},
				},
				"required": []any{"x"},
			},
			required: false,
			check: func(t *testing.T, p skill.Parameter) {
				if p.Type != "object" {
					t.Errorf("expected type 'object', got %q", p.Type)
				}
				if len(p.Properties) != 2 {
					t.Errorf("expected 2 properties, got %d", len(p.Properties))
				}
				if !p.Properties["x"].Required {
					t.Error("expected x to be required")
				}
				if p.Properties["y"].Required {
					t.Error("expected y to not be required")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := schemaToParameter(tt.schema, tt.required)
			tt.check(t, p)
		})
	}
}

func TestSkillOptions(t *testing.T) {
	sk := &SessionSkill{}

	WithSkillName("test-skill")(sk)
	if sk.name != "test-skill" {
		t.Errorf("expected name 'test-skill', got %q", sk.name)
	}

	WithSkillDescription("A test skill")(sk)
	if sk.description != "A test skill" {
		t.Errorf("expected description 'A test skill', got %q", sk.description)
	}
}

func TestSessionSkillName(t *testing.T) {
	sk := &SessionSkill{
		name:        "myskill",
		description: "My skill description",
	}

	if sk.Name() != "myskill" {
		t.Errorf("expected name 'myskill', got %q", sk.Name())
	}
	if sk.Description() != "My skill description" {
		t.Errorf("expected description 'My skill description', got %q", sk.Description())
	}
}

func TestMcpToolWrapper_NameDescription(t *testing.T) {
	wrapper := &mcpToolWrapper{
		tool: &mcp.Tool{
			Name:        "test-tool",
			Description: "A test tool",
		},
	}

	if wrapper.Name() != "test-tool" {
		t.Errorf("expected name 'test-tool', got %q", wrapper.Name())
	}
	if wrapper.Description() != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", wrapper.Description())
	}
}

func TestMcpToolWrapper_Parameters(t *testing.T) {
	wrapper := &mcpToolWrapper{
		tool: &mcp.Tool{
			Name: "param-tool",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "User name",
					},
					"age": map[string]any{
						"type":    "integer",
						"default": 0,
					},
				},
				"required": []any{"name"},
			},
		},
	}

	params := wrapper.Parameters()
	if len(params) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(params))
	}

	nameParam := params["name"]
	if nameParam.Type != "string" {
		t.Errorf("expected name type 'string', got %q", nameParam.Type)
	}
	if !nameParam.Required {
		t.Error("expected name to be required")
	}

	ageParam := params["age"]
	if ageParam.Type != "integer" {
		t.Errorf("expected age type 'integer', got %q", ageParam.Type)
	}
	if ageParam.Required {
		t.Error("expected age to not be required")
	}
}

func TestMcpToolWrapper_ParametersNil(t *testing.T) {
	wrapper := &mcpToolWrapper{
		tool: &mcp.Tool{
			Name:        "no-params",
			InputSchema: nil,
		},
	}

	params := wrapper.Parameters()
	if params != nil {
		t.Errorf("expected nil parameters, got %v", params)
	}
}
