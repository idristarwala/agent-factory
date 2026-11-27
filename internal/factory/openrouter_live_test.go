//go:build openrouter_live

package factory

import (
	"github.com/idristarwala/agent-factory/internal/configuredgateway"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveOpenRouterMixedJudges(t *testing.T) {
	gemini := os.Getenv("GEMINI_CHEAP_MODEL")
	if gemini == "" {
		gemini = "gemini-3.5-flash-lite"
	}
	router := os.Getenv("OPENROUTER_MODEL")
	if router == "" {
		router = "openai/gpt-4o"
	}
	for _, tc := range []struct{ worker, judge, workerModel, judgeModel string }{
		{"gemini", "openrouter", gemini, router},
		{"openrouter", "gemini", router, gemini},
	} {
		t.Run(tc.worker+"_worker_"+tc.judge+"_judge", func(t *testing.T) {
			t.Setenv("LLM_PROVIDER", tc.worker)
			t.Setenv("LLM_BASE_URL", "")
			t.Setenv("GEMINI_CHEAP_MODEL", gemini)
			t.Setenv("LLM_CHEAP_MODEL", router)
			t.Setenv("LLM_JUDGE_PROVIDER", tc.judge)
			t.Setenv("LLM_JUDGE_MODEL", tc.judgeModel)
			gateway, err := configuredgateway.NewFromEnv()
			if err != nil {
				t.Fatal(err)
			}
			run := testLiveFactoryRun(t, gateway)
			for i, role := range []string{"worker", "judge"} {
				call := run.Calls[i]
				want := []string{tc.workerModel, tc.judgeModel}[i]
				if call.Role != role || !strings.HasPrefix(call.Model, want) {
					t.Fatalf("expected %s model %s, got %+v", role, want, call)
				}
				t.Logf("%s: %s (%d tokens)", role, call.Model, call.Total)
			}
		})
	}
}

func TestLiveOpenRouterConfiguredJudge(t *testing.T) {
	dir := t.TempDir()
	for filename, env := range map[string]string{"google": "GEMINI_API_KEY", "router": "OPENROUTER_API_KEY"} {
		key := os.Getenv(env)
		if key == "" {
			t.Fatalf("set %s", env)
		}
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(key), 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "routing.yaml")
	const config = `gateways:
  google:
    base_url: https://generativelanguage.googleapis.com/v1beta
    auth: {type: google_api_key, key_file: google}
  router:
    base_url: https://openrouter.ai/api/v1
    auth: {type: bearer, key_file: router}
models:
  cheap: {gateway: google, protocol: gemini, model: gemini-3.5-flash-lite}
  judge: {gateway: router, protocol: openai, model: openai/gpt-4o}
`
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLM_GATEWAY_CONFIG", path)
	gateway, err := configuredgateway.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	run := testLiveFactoryRun(t, gateway)
	if run.Calls[0].Role != "worker" || run.Calls[0].Model != "gemini-3.5-flash-lite" || run.Calls[1].Role != "judge" || run.Calls[1].Model != "openai/gpt-4o" {
		t.Fatalf("incorrect model routing %+v", run.Calls)
	}
}
