package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/settings"
)

func TestContinuousSettingsCompatibility(t *testing.T) {
	t.Setenv("API_TOKEN", "")
	store := revision.NewMemory()
	mux := http.NewServeMux()
	configRoutes(mux, store)
	schema := httptest.NewRecorder()
	mux.ServeHTTP(schema, httptest.NewRequest("GET", "/api/settings/schema", nil))
	var contract struct {
		Properties map[string]struct{ Default any }
	}
	if schema.Code != 200 || json.Unmarshal(schema.Body.Bytes(), &contract) != nil {
		t.Fatal(schema.Body.String())
	}
	defaults, _, err := settings.Current(store)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(defaults)
	var values map[string]any
	if err = json.Unmarshal(raw, &values); err != nil {
		t.Fatal(err)
	}
	if len(values) != len(contract.Properties) {
		t.Fatal("schema fields differ from runtime defaults")
	}
	for k, v := range values {
		if contract.Properties[k].Default != v {
			t.Errorf("schema default mismatch for %s", k)
		}
	}
	legacy := `{"max_rounds":3,"max_minutes":5,"max_tokens":100}`
	if _, err := revision.SaveDocument(store, settings.ID, 0, "human", json.RawMessage(legacy)); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := settings.Current(store)
	if err != nil || cfg.WorkerCount != 2 || cfg.JudgeApprovals != 2 {
		t.Fatal(cfg, err)
	}
	post := func(body string, want int) {
		t.Helper()
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, httptest.NewRequest("POST", "/api/settings", strings.NewReader(body)))
		if res.Code != want {
			t.Fatalf("HTTP %d: %s", res.Code, res.Body.String())
		}
	}
	post(`{"Expected":1,"Settings":{"max_rounds":3,"max_minutes":5,"max_tokens":100,"worker_count":4,"judge_evaluations":1,"judge_approvals":2}}`, 400)
	post(`{"Expected":1,"Settings":{"max_rounds":3,"max_minutes":5,"max_tokens":100,"worker_count":4,"judge_evaluations":1,"judge_approvals":1}}`, 200)
	post(`{"Expected":2,"Settings":`+strings.Replace(legacy, "100", "101", 1)+`}`, 200)
	cfg, version, err := settings.Current(store)
	if err != nil || version != 3 || cfg.WorkerCount != 4 || cfg.JudgeEvaluations != 1 {
		t.Fatal(cfg, version, err)
	}
}
