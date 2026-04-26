// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/plexusone/omniskill/skill"
)

func TestRegisterSkill(t *testing.T) {
	rt := New(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	addTool := skill.NewTool("add", "Add two numbers",
		map[string]skill.Parameter{
			"a": {Type: "number", Description: "First number", Required: true},
			"b": {Type: "number", Description: "Second number", Required: true},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			a := params["a"].(float64)
			b := params["b"].(float64)
			return map[string]any{"sum": a + b}, nil
		},
	)

	s := &skill.BaseSkill{
		SkillName:        "math",
		SkillDescription: "Math operations",
		SkillTools:       []skill.Tool{addTool},
	}

	rt.RegisterSkill(s)

	if !rt.HasTool("add") {
		t.Error("expected tool 'add' to be registered")
	}

	if rt.ToolCount() != 1 {
		t.Errorf("expected 1 tool, got %d", rt.ToolCount())
	}
}

func TestRegisterSkillWithPrefix(t *testing.T) {
	rt := New(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	tool := skill.NewTool("greet", "Greet user", nil,
		func(ctx context.Context, params map[string]any) (any, error) {
			return "hello", nil
		},
	)

	s := &skill.BaseSkill{
		SkillName:  "greeter",
		SkillTools: []skill.Tool{tool},
	}

	rt.RegisterSkillWithPrefix(s)

	if !rt.HasTool("greeter_greet") {
		t.Error("expected tool 'greeter_greet' to be registered")
	}

	if rt.HasTool("greet") {
		t.Error("did not expect unprefixed tool 'greet'")
	}
}

func TestRegisterSkillMultipleTools(t *testing.T) {
	rt := New(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	tools := []skill.Tool{
		skill.NewTool("tool1", "Tool 1", nil, func(ctx context.Context, params map[string]any) (any, error) { return 1, nil }),
		skill.NewTool("tool2", "Tool 2", nil, func(ctx context.Context, params map[string]any) (any, error) { return 2, nil }),
		skill.NewTool("tool3", "Tool 3", nil, func(ctx context.Context, params map[string]any) (any, error) { return 3, nil }),
	}

	s := &skill.BaseSkill{
		SkillName:  "multi",
		SkillTools: tools,
	}

	rt.RegisterSkill(s)

	if rt.ToolCount() != 3 {
		t.Errorf("expected 3 tools, got %d", rt.ToolCount())
	}

	for _, tool := range tools {
		if !rt.HasTool(tool.Name()) {
			t.Errorf("expected tool %q to be registered", tool.Name())
		}
	}
}

func TestConvertToMCPTool(t *testing.T) {
	tool := skill.NewTool("test_tool", "A test tool",
		map[string]skill.Parameter{
			"name":  {Type: "string", Description: "User name", Required: true},
			"count": {Type: "integer", Description: "Count", Default: 10},
			"tags":  {Type: "array", Items: &skill.Parameter{Type: "string"}},
		},
		func(ctx context.Context, params map[string]any) (any, error) { return nil, nil },
	)

	mcpTool := convertToMCPTool(tool)

	if mcpTool.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got %q", mcpTool.Name)
	}

	if mcpTool.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", mcpTool.Description)
	}

	// Parse input schema - InputSchema is any, need to marshal/unmarshal
	schemaBytes, err := json.Marshal(mcpTool.InputSchema)
	if err != nil {
		t.Fatalf("failed to marshal schema: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("expected schema type 'object', got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in schema")
	}

	if len(props) != 3 {
		t.Errorf("expected 3 properties, got %d", len(props))
	}

	// Check required
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatal("expected required array")
	}

	if len(required) != 1 || required[0] != "name" {
		t.Errorf("expected required=['name'], got %v", required)
	}
}

func TestConvertToMCPToolNoParams(t *testing.T) {
	tool := skill.NewTool("no_params", "Tool without parameters", nil,
		func(ctx context.Context, params map[string]any) (any, error) { return nil, nil },
	)

	mcpTool := convertToMCPTool(tool)

	schemaBytes, err := json.Marshal(mcpTool.InputSchema)
	if err != nil {
		t.Fatalf("failed to marshal schema: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("expected schema type 'object', got %v", schema["type"])
	}
}

func TestParameterToSchema(t *testing.T) {
	tests := []struct {
		name  string
		param skill.Parameter
		check func(t *testing.T, schema map[string]any)
	}{
		{
			name:  "basic string",
			param: skill.Parameter{Type: "string", Description: "A string"},
			check: func(t *testing.T, schema map[string]any) {
				if schema["type"] != "string" {
					t.Errorf("expected type 'string', got %v", schema["type"])
				}
				if schema["description"] != "A string" {
					t.Errorf("expected description 'A string', got %v", schema["description"])
				}
			},
		},
		{
			name:  "with enum",
			param: skill.Parameter{Type: "string", Enum: []any{"a", "b", "c"}},
			check: func(t *testing.T, schema map[string]any) {
				enum, ok := schema["enum"].([]any)
				if !ok || len(enum) != 3 {
					t.Errorf("expected enum with 3 values, got %v", schema["enum"])
				}
			},
		},
		{
			name:  "with default",
			param: skill.Parameter{Type: "integer", Default: 42},
			check: func(t *testing.T, schema map[string]any) {
				if schema["default"] != 42 {
					t.Errorf("expected default 42, got %v", schema["default"])
				}
			},
		},
		{
			name:  "array with items",
			param: skill.Parameter{Type: "array", Items: &skill.Parameter{Type: "string"}},
			check: func(t *testing.T, schema map[string]any) {
				items, ok := schema["items"].(map[string]any)
				if !ok {
					t.Fatal("expected items object")
				}
				if items["type"] != "string" {
					t.Errorf("expected items type 'string', got %v", items["type"])
				}
			},
		},
		{
			name: "object with properties",
			param: skill.Parameter{
				Type: "object",
				Properties: map[string]skill.Parameter{
					"x": {Type: "number", Required: true},
					"y": {Type: "number"},
				},
			},
			check: func(t *testing.T, schema map[string]any) {
				props, ok := schema["properties"].(map[string]any)
				if !ok {
					t.Fatal("expected properties object")
				}
				if len(props) != 2 {
					t.Errorf("expected 2 properties, got %d", len(props))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parameterToSchema(tt.param)
			tt.check(t, schema)
		})
	}
}

func TestCreateToolHandler(t *testing.T) {
	tool := skill.NewTool("echo", "Echo input",
		map[string]skill.Parameter{
			"message": {Type: "string", Required: true},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			return map[string]any{"echoed": params["message"]}, nil
		},
	)

	handler := createToolHandler(tool)

	// Create request
	args, _ := json.Marshal(map[string]any{"message": "hello"})
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "echo",
			Arguments: args,
		},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	// Check structured content
	var output map[string]any
	if err := json.Unmarshal(result.StructuredContent.(json.RawMessage), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if output["echoed"] != "hello" {
		t.Errorf("expected echoed='hello', got %v", output["echoed"])
	}
}

func TestCreateToolHandlerError(t *testing.T) {
	tool := skill.NewTool("fail", "Always fails", nil,
		func(ctx context.Context, params map[string]any) (any, error) {
			return nil, context.DeadlineExceeded
		},
	)

	handler := createToolHandler(tool)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: "fail"},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}

	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestCallToolViaRuntime(t *testing.T) {
	rt := New(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	tool := skill.NewTool("multiply", "Multiply numbers",
		map[string]skill.Parameter{
			"a": {Type: "number", Required: true},
			"b": {Type: "number", Required: true},
		},
		func(ctx context.Context, params map[string]any) (any, error) {
			a := params["a"].(float64)
			b := params["b"].(float64)
			return map[string]any{"product": a * b}, nil
		},
	)

	s := &skill.BaseSkill{
		SkillName:  "math",
		SkillTools: []skill.Tool{tool},
	}

	rt.RegisterSkill(s)

	// Call via runtime's library mode
	result, err := rt.CallTool(context.Background(), "multiply", map[string]any{"a": 3.0, "b": 4.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse result
	var output map[string]any
	switch v := result.StructuredContent.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(v, &output); err != nil {
			t.Fatalf("failed to parse output: %v", err)
		}
	default:
		t.Fatalf("unexpected StructuredContent type: %T", result.StructuredContent)
	}

	if output["product"] != 12.0 {
		t.Errorf("expected product=12, got %v", output["product"])
	}
}
