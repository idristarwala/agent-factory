//go:build openrouter_live

package factory

import (
	"os"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/models"
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
			gateway, err := models.NewGatewayFromEnv()
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
