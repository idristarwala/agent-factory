package main

import (
	"encoding/json"
	"testing"

	"github.com/idristarwala/agent-factory/internal/factory"
)

func TestReviewArgumentsPreserveStoredNumber(t *testing.T) {
	var runs []factory.Run
	if err := json.Unmarshal([]byte(`[{"decisions":[{"proposal":{"tool_proposal":{"arguments":{"batch":[9007199254740993]}}}},{}]}]`), &runs); err != nil {
		t.Fatal(err)
	}
	if err := addArgumentText(runs); err != nil {
		t.Fatal(err)
	}
	got := runs[0].Decisions
	if got[0].ArgumentText != "{\n  \"batch\": [\n    9007199254740993\n  ]\n}" || got[1].ArgumentText != "" {
		t.Fatalf("unexpected argument text: %q, %q", got[0].ArgumentText, got[1].ArgumentText)
	}
}
