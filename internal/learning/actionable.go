package learning

// BuildActionableRule turns a failure cause into guidance without copying provider diagnostics into a prompt.
func BuildActionableRule(signal FailureSignal) string {
	switch signal.Cause {
	case "format":
		return "Return only the allowed fields as flat JSON and use the types required by the response schema."
	case "truncation":
		return "Keep the response short and within the token limit. Do not echo the full entity."
	case "provider":
		return "If the model provider is unavailable, leave the record unchanged and report the failed evaluation."
	default:
		return "When a unit cannot be evaluated, preserve its source record and report the failed step for review."
	}
}
