package main

import (
	"encoding/json"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/settings"
	"net/http"
	"os"
	"strings"
)

func configRoutes(mux *http.ServeMux, s revision.Store) {
	mux.HandleFunc("/api/releases/{id}", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		history, err := revision.Documents(s, "pack-release/"+r.PathValue("id"))
		sendJSON(w, history, err)
	}, "GET"))
	mux.HandleFunc("/api/settings/schema", apiRoute(func(w http.ResponseWriter, r *http.Request) { sendJSON(w, settings.Schema, nil) }, "GET"))
	mux.HandleFunc("/api/settings", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.URL.Query().Get("history") == "true" {
				h, err := revision.Documents(s, settings.ID)
				sendJSON(w, h, err)
				return
			}
			v, n, err := settings.Current(s)
			sendJSON(w, map[string]any{"settings": v, "version": n, "gateway": os.Getenv("LLM_PROVIDER")}, err)
			return
		}
		var edit struct {
			Expected *int
			Settings settings.Settings
		}
		current, _, err := settings.Current(s)
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		edit.Settings.ContinuousLimits = current.ContinuousLimits
		if !readEdit(w, r, &edit) {
			return
		}
		if edit.Expected == nil {
			http.Error(w, "expected revision and valid run limits are required", 400)
			return
		}
		if err := edit.Settings.Validate(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		data, _ := json.Marshal(edit.Settings)
		saved, err := revision.SaveDocument(s, settings.ID, *edit.Expected, "human", data)
		sendJSON(w, saved, err)
	}, "GET", "POST"))
	mux.HandleFunc("/api/definitions/{kind}/{id}", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		k := r.PathValue("kind")
		switch k {
		case "factory", "pack", "tool", "experience":
		default:
			http.NotFound(w, r)
			return
		}
		id := r.PathValue("id")
		if strings.TrimSpace(id) == "" || strings.Contains(id, "/") {
			http.Error(w, "invalid definition ID", 400)
			return
		}
		key := k + "/" + id
		if r.Method == "GET" {
			h, err := revision.Documents(s, key)
			sendJSON(w, h, err)
			return
		}
		var edit struct {
			Expected *int
			Data     json.RawMessage
		}
		if !readEdit(w, r, &edit) {
			return
		}
		if edit.Expected == nil || *edit.Expected < 0 || len(edit.Data) == 0 || edit.Data[0] != '{' {
			http.Error(w, "expected revision and definition object are required", 400)
			return
		}
		v, err := revision.SaveDocument(s, key, *edit.Expected, "human", edit.Data)
		sendJSON(w, v, err)
	}, "GET", "POST"))
}
