package learning

import (
	"fmt"
)

// ActionableRule builds a rule text that a worker can actually use.
// Includes the failure cause and a concrete instruction instead of only a theme name.
func ActionableRule(signal FailureSignal) string {
	switch signal.Cause {
	case "format":
		return fmt.Sprintf("Return only the allowed fields as flat JSON. Previous failure: %s", signal.Detail)
	case "truncation":
		return "Keep the response short and within the token limit. Do not echo the full entity."
	default:
		return fmt.Sprintf("Address the previous error: %s", signal.Detail)
	}
}
