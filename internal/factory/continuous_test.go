package factory

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/settings"
)

func TestContinuousUsageConcurrentWorkersAndFailedRun(t *testing.T) {
	usage := &ContinuousUsage{Gateway: scripted{}, Prices: map[string]Price{"test": {Input: 1, Output: 2}}}
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			usage.Generate(models.Request{CallerRole: "worker"})
			usage.RecordTool("source")
		})
	}
	workers.Wait()
	ws := Workspace{Store: revision.NewMemory()}
	run, err := ws.FinishContinuous("Review requests", time.Now(), settings.Settings{}, 3, usage, errors.New("round limit"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := ws.Get(run.ID)
	if err != nil || saved.Error != "round limit" || saved.Status != "failed" || saved.SettingsRevision != 3 || len(saved.Calls) != 20 || usage.Tokens() != 240 || saved.Tools["source"] != 20 {
		t.Fatal(saved, err)
	}
	for _, call := range saved.Calls {
		if call.Total != 12 || call.Cost != 0.000014 || call.Role != "worker" {
			t.Fatal(call)
		}
	}
}
