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
				RelationshipNames map[string]string      `json:"relationship_names"`
				Sample            any                    `json:"sample"`
				EditableFields    *[]string              `json:"editable_fields"`
				ProposalRevision  *int                   `json:"proposal_revision"`
				Mapping           *mapping.SourceMapping `json:"mapping"`
				Expected          *int                   `json:"expected"`
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
			if len(req.RelationshipNames) > 0 {
				if req.ProposalRevision == nil {
					http.Error(w, "relationship names require a proposal revision", 400)
					return
				}
				matched := 0
				for i, rel := range req.Mapping.Relationships {
					if name, ok := req.RelationshipNames[rel.Name]; ok {
						req.Mapping.Relationships[i].Name = name
						matched++
					}
				}
				if matched != len(req.RelationshipNames) {
					http.Error(w, "unknown proposed relationship name", 422)
					return
				}
			}
			if req.EditableFields != nil {
				if req.ProposalRevision == nil {
					http.Error(w, "editable field overrides require a proposal revision", 400)
					return
				}
				req.Mapping.EditableFields = req.EditableFields
			}
			if err := req.Mapping.ValidateEditableFields(); err != nil {
				http.Error(w, err.Error(), 422)
				return
			}
			if path := req.Mapping.NextPath; path != "" {
				value, found, err := mapping.SourceValue(req.Sample, path)
				_, isString := value.(string)
				if err != nil || !found || (value != nil && !isString) {
					http.Error(w, "next-token pointer must select a string or null in the sample", 422)
					return
				}
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
