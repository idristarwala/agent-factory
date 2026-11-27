package agents

// GuardDecision ensures a worker failure never produces an approval.
// Addresses the field-test bug where failed workers were still counted as approved.
func GuardDecision(workerErr error, decision Decision) Decision {
	if workerErr != nil {
		return Decision{
			Approved: false,
			Reason:   "worker failed: " + workerErr.Error(),
		}
	}
	return decision
}
