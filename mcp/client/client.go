// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package client provides an MCP client wrapper for connecting to MCP servers.
//
// This package simplifies connecting to MCP servers and discovering/invoking
// their tools. It wraps the official MCP SDK client with convenience methods.
//
// # Example
//
//	client := client.New("my-client", "v1.0.0", nil)
//
//	cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-github")
//	session, err := client.ConnectCommand(ctx, cmd, nil)
//	if err != nil {
//		return err
//	}
//	defer session.Close()
//
//	tools, err := session.ListTools(ctx)
//	// ...
package client

import (
	"context"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client wraps mcp.Client with convenience methods.
type Client struct {
	impl *mcp.Client
}

// Options configures a Client.
type Options struct {
	// ClientOptions are passed to the underlying mcp.Client.
	ClientOptions *mcp.ClientOptions
}

// New creates a new MCP client with the given identity.
//
// The name and version identify this client to MCP servers.
// The opts parameter may be nil to use defaults.
func New(name, version string, opts *Options) *Client {
	impl := &mcp.Implementation{
		Name:    name,
		Version: version,
	}

	var clientOpts *mcp.ClientOptions
	if opts != nil {
		clientOpts = opts.ClientOptions
	}

	return &Client{
		impl: mcp.NewClient(impl, clientOpts),
	}
}

// Connect establishes a session with an MCP server via the given transport.
//
// The transport determines how the client communicates with the server.
// Common transports include mcp.CommandTransport for spawning a subprocess.
//
// The returned Session must be closed when done.
func (c *Client) Connect(ctx context.Context, transport mcp.Transport, opts *mcp.ClientSessionOptions) (*Session, error) {
	session, err := c.impl.Connect(ctx, transport, opts)
	if err != nil {
		return nil, err
	}

	return &Session{session: session}, nil
}

// ConnectCommand spawns a command and connects to it via stdio.
//
// This is a convenience method for the common case of spawning an MCP server
// as a subprocess. The command's stdin/stdout are used for MCP communication.
//
// Environment variables can be set on the command before calling this method.
//
// Example:
//
//	cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-github")
//	cmd.Env = append(os.Environ(), "GITHUB_TOKEN=xxx")
//	session, err := client.ConnectCommand(ctx, cmd, nil)
func (c *Client) ConnectCommand(ctx context.Context, cmd *exec.Cmd, opts *mcp.ClientSessionOptions) (*Session, error) {
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	return c.Connect(ctx, transport, opts)
}

// MCPClient returns the underlying mcp.Client for advanced use cases.
func (c *Client) MCPClient() *mcp.Client {
	return c.impl
}
