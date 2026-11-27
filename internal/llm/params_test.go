package llm

import "testing"

func TestMapTokenParam(t *testing.T) {
	if got := MapTokenParam("openai", "o1-preview"); got != "max_completion_tokens" {
		t.Errorf("o1: got %s", got)
	}
	if got := MapTokenParam("openai", "gpt-5"); got != "max_completion_tokens" {
		t.Errorf("gpt-5: got %s", got)
	}
	if got := MapTokenParam("openai", "o3-mini"); got != "max_completion_tokens" {
		t.Errorf("o3: got %s", got)
	}
	if got := MapTokenParam("openai", "gpt-4o"); got != "max_tokens" {
		t.Errorf("gpt-4o: got %s", got)
	}
	if got := MapTokenParam("openai", "gpt-4-turbo"); got != "max_tokens" {
		t.Errorf("gpt-4-turbo: got %s", got)
	}
	if got := MapTokenParam("gemini", "flash"); got != "max_tokens" {
		t.Errorf("gemini: got %s", got)
	}
	if got := MapTokenParam("anthropic", "claude-3"); got != "max_tokens" {
		t.Errorf("anthropic: got %s", got)
	}
}

func TestReasoningBudget(t *testing.T) {
	if ReasoningBudget("gemini", 0) == nil {
		t.Error("expected budget map for gemini 0")
	}
	if ReasoningBudget("gemini", 1024) == nil {
		t.Error("expected budget map for gemini 1024")
	}
	if ReasoningBudget("gemini", -1) != nil {
		t.Error("expected nil for negative budget")
	}
	if ReasoningBudget("openai", 0) != nil {
		t.Error("expected nil for non-gemini")
	}
}
