package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/idristarwala/agent-factory/internal/mapping"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func sourceMappingRoutes(mux *http.ServeMux, store revision.Store) {
	handle := func(save bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			if save && (strings.TrimSpace(id) == "" || strings.Contains(id, "/")) {
				http.Error(w, "invalid mapping id", 400)
				return
			}
			key := "source-mapping/" + id
			if save && r.Method == http.MethodGet {
				h, err := revision.Documents(store, key)
				sendJSON(w, h, err)
				return
			}
			if r.Method != http.MethodPost {
				http.Error(w, "use POST", 405)
				return
			}
			var req struct {
				Sample           any                    `json:"sample"`
				ProposalRevision *int                   `json:"proposal_revision"`
				Mapping          *mapping.SourceMapping `json:"mapping"`
				Expected         *int                   `json:"expected"`
			}
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
			dec.UseNumber()
			dec.DisallowUnknownFields()
			if err := dec.Decode(&req); err != nil || dec.Decode(new(any)) != io.EOF {
				http.Error(w, "invalid mapping request", 400)
				return
			}
			if req.ProposalRevision != nil {
				n := *req.ProposalRevision
				if !save || req.Mapping != nil || n < 1 {
					http.Error(w, "select either a mapping or a proposal revision to approve", 400)
					return
				}
				h, err := revision.Documents(store, "source-proposal/"+id)
				if err != nil {
					sendJSON(w, nil, err)
					return
				}
				if n > len(h) {
					http.Error(w, "unknown proposal revision", 404)
					return
				}
				if err = json.Unmarshal(h[n-1].Data, &req.Mapping); err != nil {
					sendJSON(w, nil, err)
					return
				}
			}
			if req.Mapping == nil {
				http.Error(w, "supply a mapping", 400)
				return
			}
			rows, err := req.Mapping.Preview(req.Sample)
			if err != nil {
				http.Error(w, err.Error(), 422)
				return
			}
			valid := true
			for _, row := range rows {
				if row.Problem != "" {
					valid = false
					break
				}
			}
			if !save {
				sendJSON(w, map[string]any{"valid": valid, "rows": rows}, nil)
				return
			}
			if !valid {
				http.Error(w, "resolve preview errors before saving this mapping", 422)
				return
			}
			if req.Expected == nil || *req.Expected < 0 {
				http.Error(w, "supply expected revision", 400)
				return
			}
			observed, err := req.Mapping.Observe(req.Sample)
			if err != nil {
				http.Error(w, err.Error(), 422)
				return
			}
			data, err := json.Marshal(mapping.ApprovedMapping{SourceMapping: *req.Mapping, Observed: observed, ProposalRevision: req.ProposalRevision})
			if err != nil {
				sendJSON(w, nil, err)
				return
			}
			doc, err := revision.SaveDocument(store, key, *req.Expected, "human", data)
			sendJSON(w, doc, err)
		}
	}
	mux.HandleFunc("/api/sources/preview", apiRoute(handle(false), "POST"))
	mux.HandleFunc("/api/sources/mappings/{id}", apiRoute(handle(true), "GET", "POST"))
}
