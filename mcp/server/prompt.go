// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package runtime

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PromptHandler is an alias for the MCP SDK's prompt handler.
type PromptHandler = mcp.PromptHandler

// AddPrompt adds a prompt to the runtime.
//
// The prompt handler is called when clients request the prompt via prompts/get.
// In library mode, it can be invoked directly via [Runtime.GetPrompt].
//
// Example:
//
//	rt.AddPrompt(&mcp.Prompt{
//		Name:        "summarize",
//		Description: "Summarize the given text",
//		Arguments: []*mcp.PromptArgument{
//			{Name: "text", Description: "Text to summarize", Required: true},
//		},
//	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
//		text := req.Params.Arguments["text"]
//		return &mcp.GetPromptResult{
//			Messages: []*mcp.PromptMessage{
//				{Role: "user", Content: &mcp.TextContent{
//					Text: fmt.Sprintf("Please summarize: %s", text),
//				}},
//			},
//		}, nil
//	})
func (r *Runtime) AddPrompt(p *mcp.Prompt, h mcp.PromptHandler) {
	// Register with underlying MCP server
	r.server.AddPrompt(p, h)

	// Register in our internal map for library-mode dispatch
	r.mu.Lock()
	r.prompts[p.Name] = promptEntry{prompt: p, handler: h}
	r.mu.Unlock()
}

// RemovePrompts removes prompts with the given names from the runtime.
func (r *Runtime) RemovePrompts(names ...string) {
	r.server.RemovePrompts(names...)

	r.mu.Lock()
	for _, name := range names {
		delete(r.prompts, name)
	}
	r.mu.Unlock()
}

// HasPrompt reports whether a prompt with the given name is registered.
func (r *Runtime) HasPrompt(name string) bool {
	r.mu.RLock()
	_, ok := r.prompts[name]
	r.mu.RUnlock()
	return ok
}

// PromptCount returns the number of registered prompts.
func (r *Runtime) PromptCount() int {
	r.mu.RLock()
	n := len(r.prompts)
	r.mu.RUnlock()
	return n
}
