package agents

import (
	"strings"
)

// Clarification is a question the Steward needs answered before it can run.
type Clarification struct {
	Question string
	Field    string
	Required bool
}

// NeedsClarification checks whether the Steward has enough to proceed.
func NeedsClarification(goal string, hasSource bool, hasSample bool) []Clarification {
	var qs []Clarification

	goal = strings.TrimSpace(goal)
	if goal == "" || len(goal) < 8 {
		qs = append(qs, Clarification{
			Question: "What would you like to achieve? For example: “Fix product titles and product types.”",
			Field:    "goal",
			Required: true,
		})
	}

	if !hasSource && !hasSample {
		qs = append(qs, Clarification{
			Question: "Where does your data live? You can point me at a BigQuery table, an API, or upload a file.",
			Field:    "source",
			Required: true,
		})
	}

	lower := strings.ToLower(goal)
	if goal != "" && !strings.Contains(lower, "title") && !strings.Contains(lower, "price") &&
		!strings.Contains(lower, "inventory") && !strings.Contains(lower, "catalog") &&
		!strings.Contains(lower, "product") && !strings.Contains(lower, "stock") {
		qs = append(qs, Clarification{
			Question: "Which area should I focus on — catalog, pricing, inventory, or something else?",
			Field:    "domain",
			Required: false,
		})
	}

	return qs
}

// CanProceed is true when no required clarifications remain.
func CanProceed(qs []Clarification) bool {
	for _, q := range qs {
		if q.Required {
			return false
		}
	}
	return true
}
