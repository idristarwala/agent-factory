package agents

// WorkerConfig holds configurable limits for the LLM worker.
// Addresses the reasoning-token budget issue from the field test.
type WorkerConfig struct {
	MaxTokens int
}

// DefaultWorkerConfig returns a higher ceiling suitable for reasoning models.
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		MaxTokens: 4000, // Higher ceiling; billing only counts tokens produced.
	}
}
