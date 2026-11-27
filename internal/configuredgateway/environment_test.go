package configuredgateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/obs"
)

func TestEnvironmentWorkerUsesSharedRetriesAndSchema(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer fixture" {
			t.Error("unexpected route or auth")
		}
		var b struct {
			Model       string
			Temperature *float64
			Format      struct {
				Schema struct{ Strict *bool } `json:"json_schema"`
			} `json:"response_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Model != "small" || b.Temperature == nil || *b.Temperature != 0 || b.Format.Schema.Strict == nil || *b.Format.Schema.Strict {
			t.Errorf("unexpected model, temperature or schema mode: %+v, %v", b, err)
		}
		switch attempts {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			w.Write([]byte(`{"model":"served","choices":[{"message":{"content":"{}"}}],"usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6}}`))
		case 3:
			w.Write([]byte(`{"model":"served","choices":[{"message":{"content":"[]"}}],"usage":{"prompt_tokens":4,"completion_tokens":1}}`))
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer server.Close()
	for k, v := range map[string]string{"LLM_GATEWAY_CONFIG": "", "LLM_PROVIDER": "openai", "LLM_BASE_URL": server.URL, "LLM_API_KEY": "fixture", "LLM_CHEAP_MODEL": "small", "LLM_JUDGE_PROVIDER": "", "LLM_JUDGE_MODEL": ""} {
		t.Setenv(k, v)
	}
	gateway, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	req := models.Request{CallerRole: "worker", Tier: models.TierCheap, Schema: map[string]any{"type": "object"}, UnitID: "fixture-unit"}
	resp, err := gateway.Generate(req)
	if err != nil || resp.Model != "served" || resp.Usage["total_tokens"] != 6 || attempts != 2 {
		t.Fatal(resp, err, attempts)
	}
	invalid, err := gateway.Generate(req)
	if !errors.Is(err, llm.ErrInvalidOutput) || invalid == nil || invalid.Content != "" || invalid.Usage["total_tokens"] != 5 {
		t.Fatal(invalid, err)
	}
	_, err = gateway.Generate(req)
	var e *llm.Error
	if !errors.As(err, &e) || e.Status != 403 || attempts != 4 {
		t.Fatal(err, attempts)
	}
}

func TestEnvironmentJudgeUsesDedicatedRoute(t *testing.T) {
	for k, v := range map[string]string{"LLM_PROVIDER": "gemini", "GEMINI_API_KEY": "worker", "LLM_JUDGE_PROVIDER": "openrouter", "OPENROUTER_API_KEY": "judge", "LLM_JUDGE_MODEL": "independent"} {
		t.Setenv(k, v)
	}
	router, err := environmentRouter()
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []llm.Role{llm.RoleWorker, llm.RoleSteward, llm.RoleJudge} {
		_, wire, err := router.resolve(llm.ChatRequest{CallerRole: role, Tier: llm.TierCheap})
		if err != nil || (wire.Model == "independent") != (role == llm.RoleJudge) {
			t.Fatal(role, wire.Model, err)
		}
	}
	t.Setenv("OPENROUTER_API_KEY", "")
	if _, err := environmentRouter(); err == nil {
		t.Fatal("accepted missing dedicated key")
	}
}

func TestEnvironmentAnthropicRetainsForcedTool(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "fixture")
	p, err := providerFromEnv("anthropic")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tools  []struct{ Name string }
			Choice struct{ Name string } `json:"tool_choice"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Tools) != 1 || body.Choice.Name != "attribute_update" || r.Header.Get("x-api-key") != "fixture" {
			t.Error("missing schema enforcement or auth", err)
		}
		w.Write([]byte(`{"content":[{"type":"tool_use","name":"attribute_update","input":{"team":"IT"}}],"usage":{"input_tokens":8,"output_tokens":2}}`))
	}))
	defer server.Close()
	entry, err := buildRoute(modelConfig{Model: p.cheap, Protocol: p.protocol, SchemaMode: p.schemaMode}, server.URL, p.key, nil, server.Client(), obs.New())
	if err != nil {
		t.Fatal(err)
	}
	r := Router{routes: map[string]route{"cheap": entry}}
	resp, err := r.Generate(models.Request{CallerRole: "worker", Schema: map[string]any{"type": "object"}})
	if err != nil || resp.Content != `{"team":"IT"}` || resp.Usage["total_tokens"] != 10 {
		t.Fatal(resp, err)
	}
}
