package learning

import "github.com/idristarwala/agent-factory/internal/rules"

// ShouldActivate requires at least one newly correct case and no regressions.
func ShouldActivate(report *rules.ReplayReport) bool {
	if report == nil {
		return false
	}
	improved := false
	for _, c := range report.Cases {
		if c.BaselinePassed && !c.CandidatePassed {
			return false
		}
		if !c.BaselinePassed && c.CandidatePassed {
			improved = true
		}
	}
	return improved
}
