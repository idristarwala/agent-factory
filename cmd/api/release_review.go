package main

import (
	"encoding/json"
	"net/http"

	"github.com/idristarwala/agent-factory/internal/pack"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func releaseReviewRoutes(mux *http.ServeMux, s revision.Store) {
	mux.HandleFunc("/api/releases/{id}/active", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		history, err := revision.Documents(s, "pack-release/"+r.PathValue("id"))
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		for i := len(history) - 1; i >= 0; i-- {
			var state struct{ Status string }
			if err := json.Unmarshal(history[i].Data, &state); err != nil {
				sendJSON(w, nil, err)
				return
			}
			if state.Status == "active" {
				sendJSON(w, history[i], nil)
				return
			}
		}
		http.NotFound(w, r)
	}, "GET"))
	mux.HandleFunc("/api/releases/{id}/review", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		var edit struct {
			Expected       int
			Decision, Note string
		}
		if !readEdit(w, r, &edit) {
			return
		}
		status := map[string]string{"approve": "active", "reject": "rejected"}[edit.Decision]
		if edit.Expected < 1 || status == "" {
			http.Error(w, "supply a positive expected proposal revision and decision approve or reject", 400)
			return
		}
		key := "pack-release/" + r.PathValue("id")
		history, err := revision.Documents(s, key)
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		if len(history) == 0 {
			http.NotFound(w, r)
			return
		}
		head := history[len(history)-1]
		var release pack.Release
		if err := json.Unmarshal(head.Data, &release); err != nil {
			sendJSON(w, nil, err)
			return
		}
		if release.ProposalRevision == edit.Expected && release.Status == status && release.ReviewNote == edit.Note {
			sendJSON(w, head, nil)
			return
		}
		if head.Version != edit.Expected || release.Status != "proposed" {
			sendJSON(w, nil, revision.ErrConflict)
			return
		}
		release.Status, release.ReviewNote, release.ProposalRevision = status, edit.Note, edit.Expected
		raw, err := json.Marshal(release)
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		saved, err := revision.SaveDocument(s, key, edit.Expected, "human", raw)
		sendJSON(w, saved, err)
	}, "POST"))
}
