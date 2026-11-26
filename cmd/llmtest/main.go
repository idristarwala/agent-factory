package main

import (
	"fmt"
	"os"

	"github.com/idristarwala/agent-factory/internal/models"
)

func main() {
	gw, err := models.NewGatewayFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Set LLM_PROVIDER=gemini|openai|groq|openrouter and the matching keys\n")
		fmt.Fprintf(os.Stderr, "  gemini: GEMINI_API_KEY (or LLM_API_KEY)\n")
		fmt.Fprintf(os.Stderr, "  openai/groq/openrouter: LLM_BASE_URL + LLM_API_KEY\n")
		os.Exit(1)
	}

	resp, err := gw.Generate(models.Request{
		Tier:        models.TierCheap,
		System:      "You are a helpful assistant.",
		User:        "Reply with exactly: Agent Factory real model test OK",
		Temperature: 0.0,
		MaxTokens:   30,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Model: %s\n", resp.Model)
	fmt.Printf("Content: %s\n", resp.Content)
	fmt.Printf("Usage: %+v\n", resp.Usage)
}
