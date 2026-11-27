package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
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
