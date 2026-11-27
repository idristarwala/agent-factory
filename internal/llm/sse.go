package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// DecodeSSE reads SSE "data: {...}" lines and yields text deltas.
// Supports OpenAI, Gemini (candidates/parts/text), and Anthropic (content_block_delta) shapes.
func DecodeSSE(r io.Reader) ([]string, error) {
	var deltas []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" || data == "" {
			break
		}
		if text := extractTextDelta(data); text != "" {
			deltas = append(deltas, text)
		}
	}
	return deltas, sc.Err()
}

// extractTextDelta tries known protocol shapes and returns the first text fragment found.
func extractTextDelta(data string) string {
	// OpenAI
	var openai struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if json.Unmarshal([]byte(data), &openai) == nil {
		for _, c := range openai.Choices {
			if c.Delta.Content != "" {
				return c.Delta.Content
			}
		}
	}

	// Anthropic content_block_delta
	var anthropic struct {
		Type  string `json:"type"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}
	if json.Unmarshal([]byte(data), &anthropic) == nil && anthropic.Delta.Text != "" {
		return anthropic.Delta.Text
	}

	// Gemini candidates
	var gemini struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if json.Unmarshal([]byte(data), &gemini) == nil {
		for _, cand := range gemini.Candidates {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					return part.Text
				}
			}
		}
	}

	// Fallback string search
	if idx := strings.Index(data, `"text":"`); idx >= 0 {
		rest := data[idx+8:]
		if end := strings.Index(rest, `"`); end >= 0 {
			return rest[:end]
		}
	}
	if idx := strings.Index(data, `"content":"`); idx >= 0 {
		rest := data[idx+11:]
		if end := strings.Index(rest, `"`); end >= 0 {
			return rest[:end]
		}
	}
	return ""
}

// JoinDeltas concatenates SSE text deltas.
func JoinDeltas(deltas []string) string {
	var b bytes.Buffer
	for _, d := range deltas {
		b.WriteString(d)
	}
	return b.String()
}
