package swarm

import (
	"github.com/idristarwala/agent-factory/internal/learning"
)

// CollectFailureSignals gathers worker errors from a round so learning can act on them.
// Addresses #13 requirement that worker failures feed the learning loop.
func CollectFailureSignals(unitIDs []string, errs []error) []learning.FailureSignal {
	var signals []learning.FailureSignal
	for _, err := range errs {
		if err == nil {
			continue
		}
		signals = learning.RecordFailure(signals, err)
	}
	return signals
}
