package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// GeminiGateway uses Google's native generateContent endpoint.
// Works with free-tier models such as gemini-3.5-flash.

type GeminiGateway struct {
	APIKey        string
	CheapModel    string
	FrontierModel string
	HTTPClient    *http.Client
}

func NewGeminiFromEnv() (*GeminiGateway, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		key = os.Getenv("LLM_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY or LLM_API_KEY must be set")
	}
	cheap := os.Getenv("GEMINI_CHEAP_MODEL")
	if cheap == "" {
		cheap = "gemini-3.5-flash"
	}
	frontier := os.Getenv("GEMINI_FRONTIER_MODEL")
	if frontier == "" {
		frontier = "gemini-3.5-pro" // fallback; may need adjustment
	}
	return &GeminiGateway{
		APIKey:        key,
		CheapModel:    cheap,
		FrontierModel: frontier,
		HTTPClient:    &http.Client{Timeout: 90 * time.Second},
	}, nil
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig *struct {
		Temperature     float64 `json:"temperature,omitempty"`
		MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	} `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	ModelVersion  string `json:"modelVersion"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func (g *GeminiGateway) Generate(req Request) (*Response, error) {
	model := g.CheapModel
	switch req.Tier {
	case TierFrontier, TierCouncil:
		model = g.FrontierModel
	case TierCheap, "":
		model = g.CheapModel
	default:
		return nil, fmt.Errorf("unknown tier: %s", req.Tier)
	}

	var contents []geminiContent
	if req.System != "" {
		contents = append(contents, geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: req.System}},
		})
		contents = append(contents, geminiContent{
			Role:  "model",
			Parts: []geminiPart{{Text: "Understood."}},
		})
	}
	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: req.User}},
	})

	body := geminiRequest{Contents: contents}
	if req.Temperature > 0 || req.MaxTokens > 0 {
		body.GenerationConfig = &struct {
			Temperature     float64 `json:"temperature,omitempty"`
			MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
		}{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, g.APIKey)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode error: %w\nbody: %s", err, string(respBody))
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini error %d: %s", parsed.Error.Code, parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response: %s", string(respBody))
	}

	return &Response{
		Content: parsed.Candidates[0].Content.Parts[0].Text,
		Model:   parsed.ModelVersion,
		Usage: map[string]int{
			"prompt_tokens":     parsed.UsageMetadata.PromptTokenCount,
			"completion_tokens": parsed.UsageMetadata.CandidatesTokenCount,
			"total_tokens":      parsed.UsageMetadata.TotalTokenCount,
		},
	}, nil
}
