package swarm

import (
	"context"

	"github.com/idristarwala/agent-factory/internal/tools"
)

// GatherEvidence runs the context recipe before workers and judges.
// This wires the evidence path from issue #12 into the ContinuousRunner.
func (r *ContinuousRunner) GatherEvidence(ctx context.Context, recipe tools.ContextRecipe, registry map[string]*tools.HTTPTool, params map[string]string) (map[string]any, error) {
	return tools.ExecuteRecipe(ctx, recipe, registry, params)
}
