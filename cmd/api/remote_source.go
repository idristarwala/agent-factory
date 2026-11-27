package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/idristarwala/agent-factory/internal/mapping"
	"github.com/idristarwala/agent-factory/internal/tools"
)

type sampleSource interface {
	Sample(context.Context, string, string, map[string]string) (json.RawMessage, error)
}

func remoteSourceRoutes(mux *http.ServeMux, source sampleSource) {
	mux.HandleFunc("/api/sources/{factory}/{name}/sample", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", 405)
			return
		}
		if source == nil {
			http.Error(w, "remote sources are not configured", 503)
			return
		}
		var req struct {
			Query         map[string]string `json:"query"`
			IncludeSample bool              `json:"include_sample"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil || dec.Decode(new(any)) != io.EOF {
			http.Error(w, "invalid sampling request", 400)
			return
		}
		raw, err := source.Sample(r.Context(), r.PathValue("factory"), r.PathValue("name"), req.Query)
		if err != nil {
			switch {
			case errors.Is(err, tools.ErrSourceDenied):
				http.Error(w, "source unavailable for this factory", 403)
			case errors.Is(err, tools.ErrSourceParameter):
				http.Error(w, "unsupported source query parameter", 400)
			default:
				http.Error(w, "source sample could not be read", 502)
			}
			return
		}
		dec = json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var sample any
		if err := dec.Decode(&sample); err != nil {
			http.Error(w, "invalid upstream sample", 502)
			return
		}
		profiles, err := mapping.ProfileSample(sample)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		result := map[string]any{"collections": profiles}
		if req.IncludeSample {
			result["sample"] = raw
		}
		sendJSON(w, result, nil)
	}, "POST"))
}
