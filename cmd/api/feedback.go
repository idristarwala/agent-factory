package main

import (
	"encoding/json"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/settings"
	"net/http"
	"strings"
)

func feedbackRoute(mux *http.ServeMux, definitions revision.Store, ruleEngine *rules.Engine) {
	mux.HandleFunc("/api/feedback", apiRoute(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Note   string `json:"note"`
			Entity string `json:"entity_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Note) == "" {
			http.Error(w, "note is required", 400)
			return
		}
		cfg, _, err := settings.Current(definitions)
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		rule, err := ruleEngine.Save(rules.VersionedRule{Description: body.Note, Condition: body.Entity, Action: "constrain future proposals", Source: "feedback", CreatedBy: "customer", Active: !cfg.ReplayGate}, 0)
		if err != nil {
			sendJSON(w, nil, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "rule_created",
			"rule_id": rule.ID,
			"version": rule.Version,
			"active":  rule.Active,
			"source":  "feedback",
		})
	}, "POST"))

}
