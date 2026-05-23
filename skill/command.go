// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package skill

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandTool wraps a CLI command as a Tool.
//
// CommandTool enables SKILL.md-defined commands to be exposed as standard
// Tools, bridging the gap between markdown-based skill definitions and
// the structured Tool interface.
//
// Example:
//
//	tool := &CommandTool{
//	    ToolName:        "search",
//	    ToolDescription: "Search the archive",
//	    Command:         "notcrawl",
//	    Args:            []string{"search", "{{query}}"},
//	    ToolParameters: map[string]Parameter{
//	        "query": {Type: "string", Description: "Search query", Required: true},
//	    },
//	}
type CommandTool struct {
	// ToolName is the tool identifier.
	ToolName string

	// ToolDescription describes what the tool does.
	ToolDescription string

	// Command is the executable name or path.
	Command string

	// Args are the command arguments. Use {{paramName}} for parameter substitution.
	Args []string

	// ToolParameters defines the tool's input parameters.
	ToolParameters map[string]Parameter

	// WorkingDir is the working directory for command execution.
	// If empty, uses the current directory.
	WorkingDir string

	// Timeout is the maximum execution time. Zero means no timeout.
	Timeout time.Duration

	// Env contains additional environment variables.
	Env []string
}

// Name returns the tool name.
func (t *CommandTool) Name() string {
	return t.ToolName
}

// Description returns the tool description.
func (t *CommandTool) Description() string {
	return t.ToolDescription
}

// Parameters returns the tool parameters.
func (t *CommandTool) Parameters() map[string]Parameter {
	return t.ToolParameters
}

// Call executes the command with the given parameters.
func (t *CommandTool) Call(ctx context.Context, params map[string]any) (any, error) {
	// Apply timeout if set
	if t.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.Timeout)
		defer cancel()
	}

	// Substitute parameters in arguments
	args := make([]string, len(t.Args))
	for i, arg := range t.Args {
		args[i] = substituteParams(arg, params)
	}

	// Create command
	//nolint:gosec // G204: Command is from trusted SKILL.md tool definition
	cmd := exec.CommandContext(ctx, t.Command, args...)

	if t.WorkingDir != "" {
		cmd.Dir = t.WorkingDir
	}

	if len(t.Env) > 0 {
		cmd.Env = append(cmd.Environ(), t.Env...)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()

	result := CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("execute command %s: %w", t.Command, err)
		}
	}

	return result, nil
}

// CommandResult contains the output of a command execution.
type CommandResult struct {
	// Stdout contains the standard output.
	Stdout string `json:"stdout"`

	// Stderr contains the standard error.
	Stderr string `json:"stderr"`

	// ExitCode is the command's exit code.
	ExitCode int `json:"exit_code"`
}

// String returns the stdout if successful, or stderr if failed.
func (r CommandResult) String() string {
	if r.ExitCode == 0 {
		return strings.TrimSpace(r.Stdout)
	}
	return strings.TrimSpace(r.Stderr)
}

// substituteParams replaces {{paramName}} placeholders with parameter values.
func substituteParams(template string, params map[string]any) string {
	result := template
	for name, value := range params {
		placeholder := "{{" + name + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// NewCommandTool creates a CommandTool with common defaults.
func NewCommandTool(name, description, command string, args []string, params map[string]Parameter) *CommandTool {
	return &CommandTool{
		ToolName:        name,
		ToolDescription: description,
		Command:         command,
		Args:            args,
		ToolParameters:  params,
		Timeout:         30 * time.Second,
	}
}

// Ensure CommandTool implements Tool.
var _ Tool = (*CommandTool)(nil)
