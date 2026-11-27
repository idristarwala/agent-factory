package swarm

import (
	"github.com/idristarwala/agent-factory/internal/learning"
)

// CollectFailureSignals gathers worker errors from a round so learning can act on them.
// Addresses #13 requirement that worker failures feed the learning loop.
func CollectFailureSignals(unitIDs []string, errs []error) []learning.FailureSignal {
	var signals []learning.FailureSignal
	for i, err := range errs {
		if err == nil {
			continue
		}
		id := ""
		if i < len(unitIDs) {
			id = unitIDs[i]
		}
		signals = append(signals, learning.RecordFailure(id, err))
	}
	return signals
}
