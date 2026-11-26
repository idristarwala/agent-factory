package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// DecodeSSE reads OpenAI-style "data: {...}" lines and yields text deltas.
// Robust against whitespace and properly parses the JSON content field.
func DecodeSSE(r io.Reader) ([]string, error) {
	var deltas []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var payload struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			// Fallback to simple string search for non-standard payloads
			if idx := strings.Index(data, `"content":"`); idx >= 0 {
				rest := data[idx+11:]
				if end := strings.Index(rest, `"`); end >= 0 {
					deltas = append(deltas, rest[:end])
				}
			}
			continue
		}
		for _, c := range payload.Choices {
			if c.Delta.Content != "" {
				deltas = append(deltas, c.Delta.Content)
			}
		}
	}
	return deltas, sc.Err()
}

// JoinDeltas concatenates SSE text deltas.
func JoinDeltas(deltas []string) string {
	var b bytes.Buffer
	for _, d := range deltas {
		b.WriteString(d)
	}
	return b.String()
}
