// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package client

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Session wraps mcp.ClientSession with tool operations.
type Session struct {
	session *mcp.ClientSession
}

// ListTools returns all tools available on the server.
//
// This fetches the complete list of tools from the MCP server.
// The tools include their names, descriptions, and input schemas.
func (s *Session) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	result, err := s.session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	return result.Tools, nil
}

// CallTool invokes a tool by name with the given arguments.
//
// The args map should match the tool's input schema.
// The result contains the tool's output content.
func (s *Session) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	params := &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	return s.session.CallTool(ctx, params)
}

// GetPrompt retrieves a prompt by name with the given arguments.
func (s *Session) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	params := &mcp.GetPromptParams{
		Name:      name,
		Arguments: args,
	}

	return s.session.GetPrompt(ctx, params)
}

// ListPrompts returns all prompts available on the server.
func (s *Session) ListPrompts(ctx context.Context) ([]*mcp.Prompt, error) {
	result, err := s.session.ListPrompts(ctx, nil)
	if err != nil {
		return nil, err
	}

	return result.Prompts, nil
}

// ListResources returns all resources available on the server.
func (s *Session) ListResources(ctx context.Context) ([]*mcp.Resource, error) {
	result, err := s.session.ListResources(ctx, nil)
	if err != nil {
		return nil, err
	}

	return result.Resources, nil
}

// ReadResource reads a resource by URI.
func (s *Session) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	params := &mcp.ReadResourceParams{
		URI: uri,
	}

	return s.session.ReadResource(ctx, params)
}

// Close closes the session and releases resources.
//
// This should be called when the session is no longer needed.
// After Close, no other methods should be called on the session.
func (s *Session) Close() error {
	return s.session.Close()
}

// Wait blocks until the session is closed by the server.
//
// This is useful for long-running sessions where you want to
// wait for the server to terminate.
func (s *Session) Wait() error {
	return s.session.Wait()
}

// ID returns the session identifier.
func (s *Session) ID() string {
	return s.session.ID()
}

// InitializeResult returns the server's initialization response.
//
// This includes the server's capabilities and implementation info.
func (s *Session) InitializeResult() *mcp.InitializeResult {
	return s.session.InitializeResult()
}

// MCPSession returns the underlying mcp.ClientSession for advanced use.
func (s *Session) MCPSession() *mcp.ClientSession {
	return s.session
}
