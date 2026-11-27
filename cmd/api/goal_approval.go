package main

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/revision"
)

func goalApprovalRoute(mux *http.ServeMux, s revision.Store) {
	mux.HandleFunc("/api/chat/proposals/{id}/approve", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Version int }
		if !readEdit(w, r, &req) {
			return
		}
		id := r.PathValue("id")
		if uuid.Validate(id) != nil || req.Version < 1 {
			http.Error(w, "select a proposal revision", 400)
			return
		}
		drafts, err := revision.Documents(s, "goal-proposal/"+id)
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		// An approval consumes the next revision in the same stream as edits.
		// Its marker also allows a retry to finish creating the factory after interruption.
		retry := len(drafts) == req.Version+1 && drafts[req.Version].CreatedBy == "human"
		if req.Version < len(drafts) && !retry {
			sendJSON(w, nil, revision.ErrConflict)
			return
		}
		if req.Version > len(drafts) {
			http.Error(w, "unknown proposal revision", 404)
			return
		}
		if drafts[req.Version-1].CreatedBy != "steward" {
			sendJSON(w, nil, revision.ErrConflict)
			return
		}
		var spec factory.Spec
		if err = json.Unmarshal(drafts[req.Version-1].Data, &spec); err != nil {
			sendJSON(w, nil, err)
			return
		}
		spec.ProposalVersion = req.Version
		data, err := json.Marshal(spec)
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		if !retry {
			if _, err = revision.SaveDocument(s, "goal-proposal/"+id, req.Version, "human", data); err != nil {
				sendJSON(w, nil, err)
				return
			}
		}
		saved, err := revision.SaveDocument(s, "factory/"+id, 0, "human", data)
		if err == revision.ErrConflict {
			history, readErr := revision.Documents(s, "factory/"+id)
			if readErr != nil {
				sendJSON(w, nil, readErr)
				return
			}
			// Recover a lost approval response only while that approval is current.
			var previous factory.Spec
			if len(history) == 1 && json.Unmarshal(history[0].Data, &previous) == nil && reflect.DeepEqual(previous, spec) && history[0].CreatedBy == "human" {
				saved, err = history[0], nil
			}
		}
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		sendJSON(w, map[string]any{"factory": id, "definition": string(saved.Data), "goal": spec.Goal, "version": saved.Version}, nil)
	}, "POST"))
}
