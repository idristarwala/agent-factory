package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// streamableHTTP uses the configured evidence client's TLS, dial policy, timeout
// and rotating credentials for the entire MCP session.
type streamableHTTP struct{ tool *HTTPTool }

var _ MCPTransport = streamableHTTP{}

type mcpRoundTripper struct {
	tool     *HTTPTool
	base     http.RoundTripper
	endpoint string
}

func (t mcpRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.String() != t.endpoint {
		return nil, errors.New("unexpected MCP endpoint")
	}
	req = req.Clone(req.Context())
	if err := t.tool.applyHeaders(req); err != nil {
		return nil, err
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = http.MaxBytesReader(nil, resp.Body, maxHTTPResponse)
	return resp, nil
}

func (t streamableHTTP) CallTool(ctx context.Context, endpoint, name string, args map[string]any) (string, error) {
	client := *t.tool.Client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = mcpRoundTripper{tool: t.tool, base: base, endpoint: endpoint}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	retries := t.tool.Retries
	if retries == 0 {
		retries = -1
	}
	wire := &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: &client, MaxRetries: retries, MaxEventSize: maxHTTPResponse, DisableStandaloneSSE: true}
	peer := mcp.NewClient(&mcp.Implementation{Name: "agent-factory", Version: "1"}, nil)
	session, err := peer.Connect(ctx, wire, nil)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", errors.New("could not initialize MCP evidence session")
	}
	defer session.Close()
	if args == nil {
		args = map[string]any{}
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil || result.IsError {
		return "", errors.New("MCP evidence call failed")
	}
	if result.StructuredContent != nil {
		b, err := json.Marshal(result.StructuredContent)
		if err != nil || len(b) > maxHTTPResponse {
			return "", errors.New("invalid or oversized structured MCP evidence")
		}
		return string(b), nil
	}
	var parts []string
	for _, item := range result.Content {
		txt, ok := item.(*mcp.TextContent)
		if !ok {
			return "", errors.New("MCP evidence requires text or structured content")
		}
		parts = append(parts, txt.Text)
	}
	return strings.Join(parts, "\n"), nil
}
