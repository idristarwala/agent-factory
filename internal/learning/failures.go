package learning

import (
	"errors"
	"github.com/idristarwala/agent-factory/internal/llm"
	"strings"
)

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
	var provider *llm.Error
	switch {
	case errors.As(err, &provider) && provider.Type != "BAD_RESPONSE":
		cause = "provider"
	case strings.Contains(lower, "truncat"), strings.Contains(lower, "unexpected eof"), strings.Contains(lower, "unexpected end of json"), strings.Contains(lower, "token limit"), strings.Contains(lower, "length limit"):
		cause = "truncation"
	case strings.Contains(lower, "format"), strings.Contains(lower, "json"), strings.Contains(lower, "parse"), strings.Contains(lower, "validation"):
		cause = "format"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "429"), strings.Contains(lower, "rate limit"):
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
