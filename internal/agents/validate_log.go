package agents

import (
	"fmt"
)

// LogValidationFailure records details when structured output fails.
// Helps diagnose cut-off replies and format mismatches from the field test.
func LogValidationFailure(raw string, err error) {
	preview := raw
	if len(preview) > 300 {
		preview = preview[:300] + "..."
	}
	fmt.Printf("worker validation failed: %v | raw preview: %q | length: %d\n", err, preview, len(raw))
}
