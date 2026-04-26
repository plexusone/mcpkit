// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/plexusone/omniskill/skill"
)

// RegisterSkill registers a skill and all its tools with the runtime.
//
// Each tool from the skill is converted to an MCP tool and registered.
// The skill's Init method is NOT called - callers should initialize
// skills before registration if needed.
//
// Tool names are registered as-is (no skill prefix). If you need
// namespacing, include it in the tool names within the skill.
//
// Example:
//
//	weatherSkill := &WeatherSkill{apiKey: os.Getenv("WEATHER_API_KEY")}
//	if err := weatherSkill.Init(ctx); err != nil {
//	    return err
//	}
//	rt.RegisterSkill(weatherSkill)
func (r *Runtime) RegisterSkill(s skill.Skill) {
	for _, t := range s.Tools() {
		mcpTool := convertToMCPTool(t)
		handler := createToolHandler(t)
		r.AddToolHandler(mcpTool, handler)
	}

	r.logger.Debug("registered skill",
		"skill", s.Name(),
		"tools", len(s.Tools()),
	)
}

// RegisterSkillWithPrefix registers a skill with tool name prefixing.
//
// Each tool name is prefixed with "skillname_" to avoid conflicts.
// For example, a skill named "weather" with tool "get_forecast" would
// register as "weather_get_forecast".
func (r *Runtime) RegisterSkillWithPrefix(s skill.Skill) {
	prefix := s.Name() + "_"
	for _, t := range s.Tools() {
		mcpTool := convertToMCPTool(t)
		mcpTool.Name = prefix + mcpTool.Name
		handler := createToolHandler(t)
		r.AddToolHandler(mcpTool, handler)
	}

	r.logger.Debug("registered skill with prefix",
		"skill", s.Name(),
		"prefix", prefix,
		"tools", len(s.Tools()),
	)
}

// convertToMCPTool converts a skill.Tool to an mcp.Tool.
func convertToMCPTool(t skill.Tool) *mcp.Tool {
	mcpTool := &mcp.Tool{
		Name:        t.Name(),
		Description: t.Description(),
	}

	// Convert parameters to JSON Schema
	params := t.Parameters()
	if len(params) > 0 {
		properties := make(map[string]any)
		var required []string

		for name, param := range params {
			prop := parameterToSchema(param)
			properties[name] = prop

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

		mcpTool.InputSchema = schema
	} else {
		// Empty object schema
		mcpTool.InputSchema = map[string]any{"type": "object"}
	}

	return mcpTool
}

// parameterToSchema converts a skill.Parameter to a JSON Schema property.
func parameterToSchema(p skill.Parameter) map[string]any {
	schema := map[string]any{
		"type": p.Type,
	}

	if p.Description != "" {
		schema["description"] = p.Description
	}

	if len(p.Enum) > 0 {
		schema["enum"] = p.Enum
	}

	if p.Default != nil {
		schema["default"] = p.Default
	}

	if p.Items != nil {
		schema["items"] = parameterToSchema(*p.Items)
	}

	if len(p.Properties) > 0 {
		props := make(map[string]any)
		var required []string
		for name, prop := range p.Properties {
			props[name] = parameterToSchema(prop)
			if prop.Required {
				required = append(required, name)
			}
		}
		schema["properties"] = props
		if len(required) > 0 {
			schema["required"] = required
		}
	}

	return schema
}

// createToolHandler creates an MCP ToolHandler from a skill.Tool.
func createToolHandler(t skill.Tool) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse arguments
		var params map[string]any
		if req.Params.Arguments != nil {
			if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: "invalid arguments: " + err.Error()}},
					IsError: true,
				}, nil
			}
		}

		// Call the skill tool
		result, err := t.Call(ctx, params)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil
		}

		// Build response
		resp := &mcp.CallToolResult{}

		// Marshal result for structured content
		if result != nil {
			resultBytes, err := json.Marshal(result)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: "failed to marshal result: " + err.Error()}},
					IsError: true,
				}, nil
			}
			resp.StructuredContent = json.RawMessage(resultBytes)
			resp.Content = []mcp.Content{&mcp.TextContent{Text: string(resultBytes)}}
		}

		return resp, nil
	}
}
