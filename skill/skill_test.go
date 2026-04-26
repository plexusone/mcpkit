// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package skill

import (
	"context"
	"errors"
	"testing"
)

func TestBaseSkill(t *testing.T) {
	skill := &BaseSkill{
		SkillName:        "test",
		SkillDescription: "A test skill",
		SkillTools:       []Tool{},
	}

	if skill.Name() != "test" {
		t.Errorf("expected name 'test', got %q", skill.Name())
	}

	if skill.Description() != "A test skill" {
		t.Errorf("expected description 'A test skill', got %q", skill.Description())
	}

	if len(skill.Tools()) != 0 {
		t.Errorf("expected 0 tools, got %d", len(skill.Tools()))
	}

	if err := skill.Init(context.Background()); err != nil {
		t.Errorf("unexpected error from Init: %v", err)
	}

	if err := skill.Close(); err != nil {
		t.Errorf("unexpected error from Close: %v", err)
	}
}

func TestBaseSkillWithTools(t *testing.T) {
	tool := NewTool("greet", "Says hello", nil, func(ctx context.Context, params map[string]any) (any, error) {
		return "hello", nil
	})

	skill := &BaseSkill{
		SkillName:        "greeter",
		SkillDescription: "Greeting skill",
		SkillTools:       []Tool{tool},
	}

	tools := skill.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name() != "greet" {
		t.Errorf("expected tool name 'greet', got %q", tools[0].Name())
	}
}

func TestFuncTool(t *testing.T) {
	tool := &FuncTool{
		ToolName:        "add",
		ToolDescription: "Adds two numbers",
		ToolParameters: map[string]Parameter{
			"a": {Type: "number", Description: "First number", Required: true},
			"b": {Type: "number", Description: "Second number", Required: true},
		},
		Handler: func(ctx context.Context, params map[string]any) (any, error) {
			a := params["a"].(float64)
			b := params["b"].(float64)
			return a + b, nil
		},
	}

	if tool.Name() != "add" {
		t.Errorf("expected name 'add', got %q", tool.Name())
	}

	if tool.Description() != "Adds two numbers" {
		t.Errorf("expected description 'Adds two numbers', got %q", tool.Description())
	}

	params := tool.Parameters()
	if len(params) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(params))
	}

	if params["a"].Type != "number" {
		t.Errorf("expected parameter 'a' type 'number', got %q", params["a"].Type)
	}

	if !params["a"].Required {
		t.Error("expected parameter 'a' to be required")
	}

	result, err := tool.Call(context.Background(), map[string]any{"a": 2.0, "b": 3.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 5.0 {
		t.Errorf("expected result 5.0, got %v", result)
	}
}

func TestNewTool(t *testing.T) {
	tool := NewTool(
		"multiply",
		"Multiplies two numbers",
		map[string]Parameter{
			"x": {Type: "number", Required: true},
			"y": {Type: "number", Required: true},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			x := params["x"].(float64)
			y := params["y"].(float64)
			return x * y, nil
		},
	)

	if tool.Name() != "multiply" {
		t.Errorf("expected name 'multiply', got %q", tool.Name())
	}

	result, err := tool.Call(context.Background(), map[string]any{"x": 4.0, "y": 5.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 20.0 {
		t.Errorf("expected result 20.0, got %v", result)
	}
}

func TestFuncToolError(t *testing.T) {
	expectedErr := errors.New("tool error")
	tool := NewTool("failing", "Always fails", nil, func(ctx context.Context, params map[string]any) (any, error) {
		return nil, expectedErr
	})

	_, err := tool.Call(context.Background(), nil)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestParameterWithEnum(t *testing.T) {
	param := Parameter{
		Type:        "string",
		Description: "Temperature units",
		Enum:        []any{"celsius", "fahrenheit", "kelvin"},
		Default:     "celsius",
	}

	if param.Type != "string" {
		t.Errorf("expected type 'string', got %q", param.Type)
	}

	if len(param.Enum) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(param.Enum))
	}

	if param.Default != "celsius" {
		t.Errorf("expected default 'celsius', got %v", param.Default)
	}
}

func TestParameterWithNestedProperties(t *testing.T) {
	param := Parameter{
		Type:        "object",
		Description: "Location object",
		Properties: map[string]Parameter{
			"city":    {Type: "string", Required: true},
			"country": {Type: "string", Required: false},
		},
	}

	if param.Type != "object" {
		t.Errorf("expected type 'object', got %q", param.Type)
	}

	if len(param.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(param.Properties))
	}

	if param.Properties["city"].Type != "string" {
		t.Errorf("expected city type 'string', got %q", param.Properties["city"].Type)
	}
}

func TestParameterWithArrayItems(t *testing.T) {
	param := Parameter{
		Type:        "array",
		Description: "List of tags",
		Items:       &Parameter{Type: "string"},
	}

	if param.Type != "array" {
		t.Errorf("expected type 'array', got %q", param.Type)
	}

	if param.Items == nil {
		t.Fatal("expected Items to be non-nil")
	}

	if param.Items.Type != "string" {
		t.Errorf("expected items type 'string', got %q", param.Items.Type)
	}
}

// customSkill is a test implementation of Skill.
type customSkill struct {
	name        string
	description string
	tools       []Tool
	initCalled  bool
	closeCalled bool
	initErr     error
	closeErr    error
}

func (s *customSkill) Name() string                       { return s.name }
func (s *customSkill) Description() string                { return s.description }
func (s *customSkill) Tools() []Tool                      { return s.tools }
func (s *customSkill) Init(ctx context.Context) error     { s.initCalled = true; return s.initErr }
func (s *customSkill) Close() error                       { s.closeCalled = true; return s.closeErr }

func TestCustomSkill(t *testing.T) {
	skill := &customSkill{
		name:        "custom",
		description: "Custom skill",
		tools: []Tool{
			NewTool("echo", "Echoes input", nil, func(ctx context.Context, params map[string]any) (any, error) {
				return params["input"], nil
			}),
		},
	}

	// Verify it implements Skill
	var _ Skill = skill

	if err := skill.Init(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !skill.initCalled {
		t.Error("Init was not called")
	}

	if err := skill.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !skill.closeCalled {
		t.Error("Close was not called")
	}
}

func TestCustomSkillWithErrors(t *testing.T) {
	initErr := errors.New("init error")
	closeErr := errors.New("close error")

	skill := &customSkill{
		name:     "failing",
		initErr:  initErr,
		closeErr: closeErr,
	}

	if err := skill.Init(context.Background()); err != initErr {
		t.Errorf("expected init error %v, got %v", initErr, err)
	}

	if err := skill.Close(); err != closeErr {
		t.Errorf("expected close error %v, got %v", closeErr, err)
	}
}
