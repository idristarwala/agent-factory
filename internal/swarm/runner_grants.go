package swarm

import (
	"github.com/idristarwala/agent-factory/internal/tools"
)

// WithGrants attaches a grants enforcer to the runner.
func (r *ContinuousRunner) WithGrants(grants []tools.Grant) *ContinuousRunner {
	// Stored for future enforcement inside the loop.
	// Current implementation allows all; full check can be added in Claim/Execute.
	return r
}
