package agents

import (
	"errors"
	"fmt"
	"strconv"
)

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

var ErrInferenceConfig = errors.New("invalid agent inference configuration")

func inferenceLimits(role string, config map[string]string) (tokens, attempts int, err error) {
	tokens, attempts = DefaultWorkerConfig().MaxTokens, 2
	if role == "judge" {
		tokens, attempts = 300, 1
	}
	if value, ok := config["max_output_tokens"]; ok {
		if role != "worker" && role != "judge" {
			return 0, 0, fmt.Errorf("%w: max_output_tokens requires worker or judge role", ErrInferenceConfig)
		}
		tokens, err = strconv.Atoi(value)
		if err != nil || tokens < 1 {
			return 0, 0, fmt.Errorf("%w: max_output_tokens must be a positive integer", ErrInferenceConfig)
		}
	}
	if value, ok := config["max_attempts"]; ok {
		attempts, err = strconv.Atoi(value)
		if role != "worker" || err != nil || attempts < 1 || attempts > 16 {
			return 0, 0, fmt.Errorf("%w: max_attempts requires a worker and an integer from 1 to 16", ErrInferenceConfig)
		}
	}
	return tokens, attempts, nil
}
