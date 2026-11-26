package llm

// MapTokenParam returns the correct token limit parameter name for the model family.
// OpenAI reasoning models need max_completion_tokens; others use max_tokens.
func MapTokenParam(protocol, model string) string {
	if protocol == "openai" && contains(model, "o1") || contains(model, "gpt-5") {
		return "max_completion_tokens"
	}
	return "max_tokens"
}

// ReasoningBudget returns a thinking budget value for Gemini-style models.
func ReasoningBudget(protocol string, budget int) any {
	if protocol == "gemini" {
		return map[string]any{"thinkingBudget": budget}
	}
	return nil
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > len(sub) && (s[:len(sub)] == sub || s[len(s)-len(sub):] == sub))
}
