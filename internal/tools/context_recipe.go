package tools

import (
	"context"
	"fmt"
)

type ContextRecipe struct {
	Entity     string
	Neighbours string
	Media      string
}

func (r ContextRecipe) steps() []struct{ kind, name string } {
	return []struct{ kind, name string }{{"entity", r.Entity}, {"neighbours", r.Neighbours}}
}

// Authorize checks every requested evidence source before the first HTTP call.
func (r ContextRecipe) Authorize(pack string, grants []Grant) error {
	for _, step := range r.steps() {
		if step.name != "" && (pack == "" || !CheckGrant(grants, pack, step.name, "read-only")) {
			return fmt.Errorf("pack %q has no read-only grant for tool %q", pack, step.name)
		}
	}
	return nil
}

func ExecuteRecipe(ctx context.Context, recipe ContextRecipe, registry map[string]*HTTPTool, params map[string]string) (map[string]any, error) {
	result := make(map[string]any)
	for _, step := range recipe.steps() {
		if step.name == "" {
			continue
		}
		tool := registry[step.name]
		if tool == nil {
			return nil, fmt.Errorf("missing evidence tool %q", step.name)
		}
		body, err := tool.Call(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("evidence tool %q: %w", step.name, err)
		}
		result[step.kind] = body
	}
	return result, nil
}

// Evidence preserves the tool name and full bounded result in recipe order.
func (r ContextRecipe) Evidence(result map[string]any) []string {
	var evidence []string
	for _, step := range r.steps() {
		if value, ok := result[step.kind].(string); ok {
			evidence = append(evidence, fmt.Sprintf("%s (tool %s): %s", step.kind, step.name, value))
		}
	}
	return evidence
}
