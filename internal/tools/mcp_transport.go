package tools

import "context"

// MCPTransport is the real MCP protocol client interface.
// Implementations can use stdio, HTTP, or SSE.
type MCPTransport interface {
	CallTool(ctx context.Context, server, tool string, args map[string]any) (string, error)
}
