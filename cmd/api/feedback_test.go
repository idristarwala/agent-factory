package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/settings"
)

type rejectedRuleWrite struct{ revision.Store }

func (s rejectedRuleWrite) Insert(k revision.Kind, v revision.Snapshot) error {
	if k == revision.Rule {
		return errors.New("rule storage unavailable")
	}
	return s.Store.Insert(k, v)
}
func TestFeedbackPropagatesWriteFailure(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	st := rejectedRuleWrite{revision.NewMemory()}
	mux := http.NewServeMux()
	feedbackRoute(mux, st, rules.NewStoredEngine(st))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/feedback", strings.NewReader(`{"note":"Keep the source evidence"}`)))
	if w.Code != 500 || strings.Contains(w.Body.String(), "rule_created") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestStandaloneFeedbackUsesWorkspaceReplayGate(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	st := revision.NewMemory()
	engine := rules.NewStoredEngine(st)
	mux := http.NewServeMux()
	feedbackRoute(mux, st, engine)
	for i, gated := range []bool{false, true} {
		cfg := settings.Settings{MaxRounds: 1, MaxTokens: 100, MaxMinutes: 1, ReplayGate: gated}
		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = revision.SaveDocument(st, settings.ID, i, "human", data); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/feedback", strings.NewReader(`{"note":"Use the routing guide"}`)))
		var result struct {
			ID     string `json:"rule_id"`
			Active bool
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || result.ID == "" || result.Active == gated {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
		h, err := engine.List(result.ID)
		if err != nil || len(h) != 1 || h[0].Active != result.Active || h[0].Version != 1 {
			t.Fatal(h, err)
		}
	}
}
