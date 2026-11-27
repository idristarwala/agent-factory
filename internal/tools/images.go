package tools

import (
	"context"
	"io"
	"net/http"
)

// FetchImages downloads image URLs for multimodal models.
// Subject to size limits. Returns base64 or raw bytes for gateway adapters.
func FetchImages(ctx context.Context, urls []string, maxKB int) ([][]byte, error) {
	var results [][]byte
	client := &http.Client{}
	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxKB*1024)))
		resp.Body.Close()
		results = append(results, body)
	}
	return results, nil
}
