package tools

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/idristarwala/agent-factory/internal/llm"
)

// RecordEvidence holds the inputs shared by the worker and synchronous judge.
type RecordEvidence struct {
	Text   []string
	Images []llm.ImagePart
}

func (r ContextRecipe) collect(ctx context.Context, result map[string]any, client *http.Client) (RecordEvidence, error) {
	out := RecordEvidence{Text: r.Evidence(result)}
	if r.Media == "" {
		return out, nil
	}
	body, ok := result["entity"].(string)
	if !ok {
		return RecordEvidence{}, fmt.Errorf("media requires entity evidence")
	}
	var record map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &record); err != nil {
		return RecordEvidence{}, fmt.Errorf("media source is not a JSON object")
	}
	var urls []string
	field, ok := record[r.Media]
	if !ok || string(field) == "null" || json.Unmarshal(field, &urls) != nil {
		return RecordEvidence{}, fmt.Errorf("media field %q requires an array of image URLs", r.Media)
	}
	if len(urls) > 10 {
		return RecordEvidence{}, fmt.Errorf("media field exceeds 10 images per record")
	}
	for i, url := range urls {
		part, hash, err := readRecordImage(ctx, client, url)
		if err != nil {
			return RecordEvidence{}, fmt.Errorf("media from tool %q at index %d: %w", r.Entity, i, err)
		}
		out.Images = append(out.Images, part)
		out.Text = append(out.Text, fmt.Sprintf("image %d (tool %s, field %s): sha256 %x", i+1, r.Entity, r.Media, hash))
	}
	return out, nil
}

func readRecordImage(ctx context.Context, client *http.Client, url string) (llm.ImagePart, [32]byte, error) {
	var empty llm.ImagePart
	var hash [32]byte
	if validateDestination(url, url) != nil {
		return empty, hash, fmt.Errorf("invalid image destination")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return empty, hash, fmt.Errorf("invalid image request")
	}
	resp, err := client.Do(req)
	if err != nil {
		return empty, hash, fmt.Errorf("image request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return empty, hash, fmt.Errorf("image source returned status %d", resp.StatusCode)
	}
	const limit = 512 << 10
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(b) > limit {
		return empty, hash, fmt.Errorf("unable to read image within 512 KiB limit")
	}
	mime := http.DetectContentType(b)
	switch mime {
	case "image/png", "image/jpeg", "image/webp":
	default:
		return empty, hash, fmt.Errorf("unsupported image content")
	}
	return llm.ImagePart{Mime: mime, Base64: base64.StdEncoding.EncodeToString(b)}, sha256.Sum256(b), nil
}

func (r ContextRecipe) Collect(ctx context.Context, result map[string]any) (RecordEvidence, error) {
	if r.Media == "" {
		return RecordEvidence{Text: r.Evidence(result)}, nil
	}
	client := publicClient()
	defer client.CloseIdleConnections()
	return r.collect(ctx, result, client)
}
