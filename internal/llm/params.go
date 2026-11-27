package llm

import "strings"

// MapTokenParam returns the correct token limit parameter name for the model family.
// OpenAI reasoning models need max_completion_tokens; others use max_tokens.
func MapTokenParam(protocol, model string) string {
	if protocol == "openai" && (strings.Contains(model, "o1") || strings.Contains(model, "gpt-5") || strings.Contains(model, "o3")) {
		return "max_completion_tokens"
	}
	return "max_tokens"
}

// ReasoningBudget returns a thinking budget value for Gemini-style models.
// budget < 0 means omit; 0 disables thinking.
func ReasoningBudget(protocol string, budget int) any {
	if protocol == "gemini" && budget >= 0 {
		return map[string]any{"thinkingBudget": budget}
	}
	return nil
}
