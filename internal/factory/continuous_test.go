package factory

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/obs"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/settings"
)

func TestContinuousUsageConcurrentWorkersAndFailedRun(t *testing.T) {
	usage := &ContinuousUsage{Gateway: scripted{}, Prices: map[string]Price{"test": {Input: 1, Output: 2}}}
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			usage.Generate(models.Request{CallerRole: "worker", UnitID: "unit-1"})
			usage.RecordTool("source")
		})
	}
	workers.Wait()
	ws := Workspace{Store: revision.NewMemory()}
	audit := []obs.AuditEntry{{TraceID: "unit-1", Actor: "worker", Action: "proposed"}}
	run, err := ws.FinishContinuous("Review requests", time.Now(), settings.Settings{}, 3, usage, audit, errors.New("round limit"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := ws.Get(run.ID)
	if err != nil || saved.Error != "round limit" || saved.Status != "failed" || saved.SettingsRevision != 3 || len(saved.Calls) != 20 || usage.Tokens() != 240 || saved.Tools["source"] != 20 {
		t.Fatal(saved, err)
	}
	if len(saved.Audit) != 1 || saved.Audit[0].Action != "proposed" {
		t.Fatal(saved.Audit)
	}
	for _, call := range saved.Calls {
		if call.UnitID != saved.Audit[0].TraceID || call.Total != 12 || call.Cost != 0.000014 || call.Role != "worker" {
			t.Fatal(call)
		}
	}
}
