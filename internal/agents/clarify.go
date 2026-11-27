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
			Question: "What outcome should this factory achieve?",
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
