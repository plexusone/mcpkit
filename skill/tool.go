// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package skill

import "context"

// Tool represents a single callable function within a skill.
//
// Tools are the atomic unit of functionality. Each tool has:
//   - A unique name within its skill
//   - A description for AI models to understand its purpose
//   - Parameters describing its inputs
//   - A handler that executes the tool logic
//
// Example implementation:
//
//	type GetWeatherTool struct{}
//
//	func (t *GetWeatherTool) Name() string        { return "get_current_weather" }
//	func (t *GetWeatherTool) Description() string { return "Get the current weather for a location" }
//	func (t *GetWeatherTool) Parameters() map[string]Parameter {
//	    return map[string]Parameter{
//	        "location": {Type: "string", Description: "City name", Required: true},
//	        "units":    {Type: "string", Description: "Temperature units", Enum: []any{"celsius", "fahrenheit"}},
//	    }
//	}
//	func (t *GetWeatherTool) Call(ctx context.Context, params map[string]any) (any, error) {
//	    location := params["location"].(string)
//	    // ... fetch weather ...
//	    return map[string]any{"temperature": 72, "condition": "sunny"}, nil
//	}
type Tool interface {
	// Name returns the tool identifier (e.g., "get_current_weather").
	// Names should be lowercase, alphanumeric with underscores.
	Name() string

	// Description returns a human-readable description of what the tool does.
	// This is used by AI models to understand when to use the tool.
	Description() string

	// Parameters returns the JSON Schema-like parameter definitions.
	// The map keys are parameter names.
	Parameters() map[string]Parameter

	// Call executes the tool with the given parameters.
	// Parameters have already been validated against the schema.
	// Returns the tool output or an error.
	Call(ctx context.Context, params map[string]any) (any, error)
}

// Parameter describes a tool parameter.
//
// Parameters follow JSON Schema conventions for describing input types.
type Parameter struct {
	// Type is the JSON Schema type: "string", "number", "integer", "boolean", "object", "array".
	Type string `json:"type"`

	// Description explains what the parameter is for.
	Description string `json:"description,omitempty"`

	// Required indicates whether the parameter must be provided.
	Required bool `json:"required,omitempty"`

	// Enum lists the allowed values for this parameter.
	Enum []any `json:"enum,omitempty"`

	// Default is the default value if not provided.
	Default any `json:"default,omitempty"`

	// Items describes array element type (when Type is "array").
	Items *Parameter `json:"items,omitempty"`

	// Properties describes object properties (when Type is "object").
	Properties map[string]Parameter `json:"properties,omitempty"`
}

// ToolFunc is a function type that can be used as a tool handler.
type ToolFunc func(ctx context.Context, params map[string]any) (any, error)

// FuncTool wraps a function as a Tool.
// This is useful for creating simple tools without implementing the full interface.
type FuncTool struct {
	ToolName        string
	ToolDescription string
	ToolParameters  map[string]Parameter
	Handler         ToolFunc
}

// Name returns the tool name.
func (t *FuncTool) Name() string {
	return t.ToolName
}

// Description returns the tool description.
func (t *FuncTool) Description() string {
	return t.ToolDescription
}

// Parameters returns the tool parameters.
func (t *FuncTool) Parameters() map[string]Parameter {
	return t.ToolParameters
}

// Call executes the tool handler.
func (t *FuncTool) Call(ctx context.Context, params map[string]any) (any, error) {
	return t.Handler(ctx, params)
}

// Ensure FuncTool implements Tool.
var _ Tool = (*FuncTool)(nil)

// NewTool creates a new FuncTool with the given properties.
func NewTool(name, description string, params map[string]Parameter, handler ToolFunc) *FuncTool {
	return &FuncTool{
		ToolName:        name,
		ToolDescription: description,
		ToolParameters:  params,
		Handler:         handler,
	}
}

// ToJSONSchema converts the tool's parameters to JSON Schema format
// compatible with OpenAI/Anthropic function calling.
func (t *FuncTool) ToJSONSchema() map[string]any {
	return ParametersToJSONSchema(t.ToolParameters)
}

// ParametersToJSONSchema converts a parameter map to JSON Schema format.
func ParametersToJSONSchema(params map[string]Parameter) map[string]any {
	properties := make(map[string]any)
	var required []string

	for name, param := range params {
		properties[name] = ParameterToSchema(param)
		if param.Required {
			required = append(required, name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// ParameterToSchema converts a Parameter to JSON Schema format.
func ParameterToSchema(p Parameter) map[string]any {
	schema := map[string]any{
		"type": p.Type,
	}

	if p.Description != "" {
		schema["description"] = p.Description
	}

	if p.Default != nil {
		schema["default"] = p.Default
	}

	if len(p.Enum) > 0 {
		schema["enum"] = p.Enum
	}

	if p.Items != nil {
		schema["items"] = ParameterToSchema(*p.Items)
	}

	if len(p.Properties) > 0 {
		props := make(map[string]any)
		var req []string
		for name, prop := range p.Properties {
			props[name] = ParameterToSchema(prop)
			if prop.Required {
				req = append(req, name)
			}
		}
		schema["properties"] = props
		if len(req) > 0 {
			schema["required"] = req
		}
	}

	return schema
}
