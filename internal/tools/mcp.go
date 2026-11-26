package tools

import (
	"context"
	"fmt"
)

// MCPTool wraps any MCP server tool so agents can call it.
// Real implementation would use an MCP client; this is the seam.
type MCPTool struct {
	ServerName string
	ToolName   string
	Desc       string
	Schema     map[string]any
	// CallFn is injected so tests and different transports can plug in.
	CallFn func(ctx context.Context, args map[string]any) (string, error)
}

func (t *MCPTool) Name() string        { return t.ServerName + "." + t.ToolName }
func (t *MCPTool) Description() string { return t.Desc }
func (t *MCPTool) Parameters() map[string]any {
	if t.Schema != nil {
		return t.Schema
	}
	return map[string]any{"type": "object"}
}

func (t *MCPTool) Call(ctx context.Context, args map[string]any) (string, error) {
	if t.CallFn == nil {
		return "", fmt.Errorf("MCP tool %s has no CallFn", t.Name())
	}
	return t.CallFn(ctx, args)
}
