package learning

import (
	"fmt"
)

// BuildActionableRule builds a rule text from a structured failure signal.
// Includes the failure cause and a concrete instruction instead of only a theme name.
func BuildActionableRule(signal FailureSignal) string {
	switch signal.Cause {
	case "format":
		return fmt.Sprintf("Return only the allowed fields as flat JSON. Previous failure: %s", signal.Detail)
	case "truncation":
		return "Keep the response short and within the token limit. Do not echo the full entity."
	default:
		return fmt.Sprintf("Address the previous error: %s", signal.Detail)
	}
}
