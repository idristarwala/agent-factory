package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRecordMedia(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Write(buf.Bytes())
		case "/large":
			w.Write(make([]byte, 512*1024+1))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	recipe := ContextRecipe{Entity: "source", Media: "files"}
	for _, tc := range []struct {
		urls []string
		ok   bool
	}{
		{[]string{srv.URL + "/ok"}, true}, {[]string{}, true}, {[]string{srv.URL + "/missing"}, false}, {[]string{srv.URL + "/large"}, false}, {make([]string, 11), false}, {nil, false},
	} {
		record, _ := json.Marshal(map[string]any{"files": tc.urls})
		got, err := recipe.collect(context.Background(), map[string]any{"entity": string(record)}, srv.Client())
		if (err == nil) != tc.ok {
			t.Fatalf("unexpected error %v", err)
		}
		if tc.ok && len(tc.urls) > 0 && (len(got.Images) != 1 || got.Images[0].Base64 != base64.StdEncoding.EncodeToString(buf.Bytes()) || !strings.Contains(got.Text[1], "tool source, field files")) {
			t.Fatal("missing image or provenance")
		}
	}
}

func TestLargeImageSetPreservesFinalSource(t *testing.T) {
	var first, last bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	if err := png.Encode(&first, img); err != nil {
		t.Fatal(err)
	}
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
	if err := png.Encode(&last, img); err != nil {
		t.Fatal(err)
	}
	requests := atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path == "/last" {
			w.Write(last.Bytes())
			return
		}
		w.Write(first.Bytes())
	}))
	defer srv.Close()
	run := func(n int) (RecordEvidence, error) {
		urls := make([]string, n)
		for i := range urls {
			urls[i] = srv.URL
		}
		urls[n-1] += "/last"
		record, _ := json.Marshal(map[string]any{"files": urls})
		return (ContextRecipe{Entity: "source", Media: "files"}).collect(context.Background(), map[string]any{"entity": string(record)}, srv.Client())
	}
	evidence, err := run(51)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Images) != 9 || requests.Load() != 51 {
		t.Fatalf("images %d, requests %d", len(evidence.Images), requests.Load())
	}
	note := fmt.Sprintf("image 51 (tool source, field files): sha256 %x (sheet 9, row 2, column 1)", sha256.Sum256(last.Bytes()))
	if !slices.Contains(evidence.Text, note) {
		t.Fatal("missing provenance for final tile")
	}
	raw, err := base64.StdEncoding.DecodeString(evidence.Images[8].Base64)
	if err != nil {
		t.Fatal(err)
	}
	result, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := result.At(160, 480).RGBA()
	if r < 60000 || g > 1000 || b > 1000 {
		t.Fatal("last tile missing or misplaced")
	}
	if _, err := run(91); err == nil || requests.Load() != 51 {
		t.Fatal("oversized sets must fail before a fetch")
	}
}
