package tools

import "context"

// Tool is anything an agent can call during a ReAct loop.
// Implementations: MCP tools, SQL, BigQuery, HTTP, graph queries, etc.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any // JSON schema
	Call(ctx context.Context, args map[string]any) (string, error)
}

// Registry holds available tools.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

func (r *Registry) Specs() []ToolSpec {
	var specs []ToolSpec
	for _, t := range r.tools {
		specs = append(specs, ToolSpec{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return specs
}

type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any
}
