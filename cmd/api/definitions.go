package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
)

func sendJSON(w http.ResponseWriter, value any, err error) {
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, agents.ErrInferenceConfig) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, revision.ErrConflict) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}
func readEdit(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		http.Error(w, "Invalid definition edit", http.StatusBadRequest)
		return false
	}
	return true
}
func definitionRoutes(mux *http.ServeMux, engine *rules.Engine, registry *agents.Registry) {
	mux.HandleFunc("/api/rules", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("history") == "true" {
				v, err := engine.List(r.URL.Query().Get("id"))
				sendJSON(w, v, err)
				return
			}
			v, err := engine.Current()
			sendJSON(w, v, err)
			return
		}
		var edit struct {
			ID, Description, Condition, Action string
			Expected                           *int
			Promote                            int
		}
		if !readEdit(w, r, &edit) {
			return
		}
		if edit.Expected == nil || *edit.Expected < 0 {
			http.Error(w, "expected revision is required", 400)
			return
		}
		if edit.Promote != 0 {
			if edit.ID == "" || edit.Promote < 1 || edit.Promote > *edit.Expected {
				http.Error(w, "invalid promotion", 400)
				return
			}
			v, err := engine.Promote(edit.ID, edit.Promote, *edit.Expected, "human")
			sendJSON(w, v, err)
			return
		}
		if strings.TrimSpace(edit.Description) == "" {
			http.Error(w, "description is required", 400)
			return
		}
		v, err := engine.Save(rules.VersionedRule{ID: edit.ID, Description: edit.Description, Condition: edit.Condition, Action: edit.Action, CreatedBy: "human", Source: "human"}, *edit.Expected)
		sendJSON(w, v, err)
	}, "GET", "POST"))
	mux.HandleFunc("/api/agents", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			v, err := registry.History(r.URL.Query().Get("name"))
			sendJSON(w, v, err)
			return
		}
		var edit struct {
			Name, Role, Version, ModelTier string
			Config                         map[string]string
			Tools                          []string
			Retired                        bool
			Expected                       *int
		}
		if !readEdit(w, r, &edit) {
			return
		}
		if strings.TrimSpace(edit.Name) == "" || strings.TrimSpace(edit.Role) == "" || edit.Expected == nil || *edit.Expected < 0 {
			http.Error(w, "name, role and expected revision are required", 400)
			return
		}
		v, err := registry.Save(agents.AgentVersion{Name: edit.Name, Role: edit.Role, Version: edit.Version, ModelTier: edit.ModelTier, Config: edit.Config, Tools: edit.Tools, Retired: edit.Retired, CreatedBy: "human"}, *edit.Expected)
		sendJSON(w, v, err)
	}, "GET", "POST"))
}
