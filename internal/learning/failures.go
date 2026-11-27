package learning

import "strings"

// FailureSignal is a worker failure that the learning loop can see.
type FailureSignal struct {
	Cause  string // "format", "truncation", "provider", "other"
	Detail string
	Count  int
}

// RecordFailure groups worker failures by cause.
func RecordFailure(signals []FailureSignal, err error) []FailureSignal {
	if err == nil {
		return signals
	}
	msg := err.Error()
	cause := "other"
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "json"), strings.Contains(lower, "parse"), strings.Contains(lower, "validation"):
		cause = "format"
	case strings.Contains(lower, "truncat"), strings.Contains(lower, "token"), strings.Contains(lower, "length"):
		cause = "truncation"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "429"), strings.Contains(lower, "rate"):
		cause = "provider"
	}
	for i := range signals {
		if signals[i].Cause == cause {
			signals[i].Count++
			signals[i].Detail = msg
			return signals
		}
	}
	return append(signals, FailureSignal{Cause: cause, Detail: msg, Count: 1})
}
