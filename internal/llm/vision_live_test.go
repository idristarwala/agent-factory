//go:build openrouter_live

package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLiveOpenRouterMixedVision(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	for y := 0; y < 96; y++ {
		for x := 0; x < 96; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 240, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	part := ImagePart{Mime: "image/png", Base64: base64.StdEncoding.EncodeToString(buf.Bytes())}
	for _, tc := range []struct {
		provider, key, model string
		role                 Role
	}{
		{"gemini", "GEMINI_API_KEY", "gemini-3.5-flash-lite", RoleWorker}, {"openrouter", "OPENROUTER_API_KEY", "openai/gpt-4o", RoleJudge},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			key := os.Getenv(tc.key)
			if key == "" {
				t.Fatalf("set %s", tc.key)
			}
			gw, err := New(tc.provider, map[string]string{"api_key": key, "cheap_model": tc.model})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			reply, err := gw.Chat(ctx, ChatRequest{CallerRole: tc.role, Model: tc.model, MaxTokens: 150, Messages: []Message{{Role: "user", Content: "What color fills this image? Answer with one color word.", Images: []ImagePart{part}}}})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Trim(strings.ToLower(strings.TrimSpace(reply.Content)), ".") != "red" || reply.Usage.Total == 0 {
				t.Fatalf("unexpected color %q or absent usage", reply.Content)
			}
		})
	}
}
