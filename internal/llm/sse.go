package llm

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// DecodeSSE reads OpenAI-style "data: {...}" lines and yields text deltas.
func DecodeSSE(r io.Reader) ([]string, error) {
	var deltas []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		if idx := strings.Index(data, `"content":`); idx >= 0 {
			rest := data[idx+10:]
			if end := strings.Index(rest, `"`); end > 0 {
				deltas = append(deltas, rest[1:end])
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
