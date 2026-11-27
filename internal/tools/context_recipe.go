package tools

import (
	"context"
)

// ContextRecipe declares what evidence to gather before the worker and judge run.
// From issue #12.
type ContextRecipe struct {
	Entity     string // tool name for the main entity
	Neighbours string // tool name for family/search
	Media      string // field containing image URLs
}

// ExecuteRecipe runs the declared tools and returns the gathered context.
func ExecuteRecipe(ctx context.Context, recipe ContextRecipe, registry map[string]*HTTPTool, params map[string]string) (map[string]any, error) {
	out := make(map[string]any)
	if recipe.Entity != "" {
		if tool, ok := registry[recipe.Entity]; ok {
			body, err := tool.Call(ctx, params)
			if err == nil {
				out["entity"] = body
			}
		}
	}
	if recipe.Neighbours != "" {
		if tool, ok := registry[recipe.Neighbours]; ok {
			body, err := tool.Call(ctx, params)
			if err == nil {
				out["neighbours"] = body
			}
		}
	}
	return out, nil
}
