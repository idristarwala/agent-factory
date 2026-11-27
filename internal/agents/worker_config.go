package agents

// WorkerConfig holds configurable limits for the LLM worker.
// Addresses the reasoning-token budget and truncated JSON issues from the field test.
type WorkerConfig struct {
	MaxTokens int
}

// DefaultWorkerConfig returns a high ceiling suitable for reasoning models and rule-heavy prompts.
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		MaxTokens: 16000, // High ceiling so thinking tokens and full JSON do not truncate.
	}
}
