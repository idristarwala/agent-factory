package swarm

import (
	"context"

	"github.com/idristarwala/agent-factory/internal/tools"
)

// GatherEvidence runs the context recipe before workers and judges.
// This wires the evidence path from issue #12 into the ContinuousRunner.
func (r *ContinuousRunner) GatherEvidence(ctx context.Context, recipe tools.ContextRecipe, registry map[string]*tools.HTTPTool, params map[string]string) (map[string]any, error) {
	if err := recipe.Authorize(r.Pack, r.Grants); err != nil {
		return nil, err
	}
	var record func(string)
	if r.Usage != nil {
		record = r.Usage.RecordTool
	}
	return tools.ExecuteRecipeTracked(ctx, recipe, registry, params, record)
}
