package tools

import (
	"context"
)

// ImageEvidence fetches a limited set of image URLs and returns summaries for prompts.
// Full binary attachment to gateway adapters is handled by ImagePart in the llm package.
func ImageEvidence(ctx context.Context, urls []string, max int) []string {
	if max <= 0 || max > 50 {
		max = 10
	}
	if len(urls) > max {
		urls = urls[:max]
	}
	imgs, _ := FetchImages(ctx, urls, 256)
	var out []string
	for i, img := range imgs {
		out = append(out, fmt.Sprintf("image_%d: %d bytes", i, len(img)))
	}
	return out
}
