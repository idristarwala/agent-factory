package swarm

import (
	"github.com/idristarwala/agent-factory/internal/rules"
)

// ActiveRulesForRound returns the current active rules so a new worker can be
// built with them. Call this each round before CreateWorker.
// Addresses the last gap in #13.
func ActiveRulesForRound(engine *rules.Engine) []rules.VersionedRule {
	if engine == nil {
		return nil
	}
	return engine.Active()
}
