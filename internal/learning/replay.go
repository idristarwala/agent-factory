package learning

// ShouldActivate is a simple replay gate. In a full implementation this would
// re-run a held-out sample. For now it activates only if the signal count is low
// or the cause is understood.
func ShouldActivate(signals []FailureSignal) bool {
	if len(signals) == 0 {
		return true
	}
	// Activate format and truncation fixes immediately; hold others for review.
	for _, s := range signals {
		if s.Cause == "format" || s.Cause == "truncation" {
			return true
		}
	}
	return false
}
