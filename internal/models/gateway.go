package models

import "github.com/idristarwala/agent-factory/internal/llm"

// ModelGateway is the single entry point for all LLM calls.
// Workers are restricted to cheap models; Steward and Judge may use frontier or council.

type Tier string

const (
	TierCheap    Tier = "cheap"    // Gemini Flash, DeepSeek Flash, Groq, etc.
	TierFrontier Tier = "frontier" // highest quality, restricted to Steward / Judge
	TierCouncil  Tier = "council"  // optional multi-model for Judge
)

type Request struct {
	UnitID      string // Local audit correlation, never sent to a provider.
	CallerRole  string
	Tier        Tier
	System      string
	User        string
	Images      []llm.ImagePart
	Temperature float64
	MaxTokens   int
	Tools       []string
	Schema      map[string]any // optional JSON schema for provider-enforced output
}

type Response struct {
	Content string
	Model   string
	Usage   map[string]int
}

type Gateway interface {
	Generate(req Request) (*Response, error)
}

// Provider-agnostic. Concrete implementations (OpenAI-compatible, Gemini, Anthropic, etc.)
// live behind this interface and enforce tier rules.
