package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/tools"
)

// Expose a stable upstream failure without returning remote URLs or credential paths.
type importFeed struct{ sampleSource }

var errImportFeed = errors.New("unable to fetch source page")

func (f importFeed) Sample(ctx context.Context, factory, name string, q map[string]string) (json.RawMessage, error) {
	raw, err := f.sampleSource.Sample(ctx, factory, name, q)
	if err != nil && !errors.Is(err, tools.ErrSourceDenied) && !errors.Is(err, tools.ErrSourceParameter) {
		return nil, errImportFeed
	}
	return raw, err
}
func pagedImportRoutes(mux *http.ServeMux, ws *factory.Workspace, source sampleSource) {
	mux.HandleFunc("/api/imports/{factory}/{id}", operatorRoute(func(w http.ResponseWriter, r *http.Request) {
		name, id := r.PathValue("factory"), r.PathValue("id")
		if r.Method == http.MethodGet {
			pages, err := ws.ImportPages(name, id)
			sendJSON(w, pages, err)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "use GET or POST", 405)
			return
		}
		if source == nil {
			http.Error(w, "remote sources are not configured", 503)
			return
		}
		var req struct {
			Spec     factory.ImportSpec
			Expected *int
			MaxPages int `json:"max_pages"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil || req.Expected == nil || dec.Decode(new(any)) != io.EOF {
			http.Error(w, "invalid import request or missing expected checkpoint", 400)
			return
		}
		if req.MaxPages == 0 {
			req.MaxPages = 1
		}
		pages, err := ws.AdvanceImport(r.Context(), importFeed{source}, name, id, req.Spec, *req.Expected, req.MaxPages)
		if err != nil {
			status := 422
			if errors.Is(err, errImportFeed) {
				status = 502
			}
			if errors.Is(err, revision.ErrConflict) {
				status = 409
			}
			if errors.Is(err, tools.ErrSourceDenied) {
				status = 403
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]any{"committed_pages": len(pages), "error": err.Error()})
			return
		}
		sendJSON(w, pages, nil)
	}, "GET", "POST"))
}
