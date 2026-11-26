package tools

import (
	"context"
	"fmt"
)

// MCPTransport is the real MCP protocol client interface.
// Implementations can use stdio, HTTP, or SSE.
type MCPTransport interface {
	CallTool(ctx context.Context, server, tool string, args map[string]any) (string, error)
}

// MCPClientTool uses a real transport.
type MCPClientTool struct {
	Server    string
	Tool      string
	Desc      string
	Schema    map[string]any
	Transport MCPTransport
}

func (t *MCPClientTool) Name() string        { return t.Server + "." + t.Tool }
func (t *MCPClientTool) Description() string { return t.Desc }
func (t *MCPClientTool) Parameters() map[string]any {
	if t.Schema != nil {
		return t.Schema
	}
	return map[string]any{"type": "object"}
}

func (t *MCPClientTool) Call(ctx context.Context, args map[string]any) (string, error) {
	if t.Transport == nil {
		return "", fmt.Errorf("MCP transport not configured for %s", t.Name())
	}
	return t.Transport.CallTool(ctx, t.Server, t.Tool, args)
}
