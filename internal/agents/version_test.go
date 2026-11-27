package agents

import (
	"errors"
	"github.com/idristarwala/agent-factory/internal/revision"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDefinitionHistoryAcrossStewardAndHumanEdits(t *testing.T) {
	r := NewRegistry()
	base := AgentVersion{Name: "triage", Role: "worker", ModelTier: "cheap", Config: map[string]string{"instructions": "original"}, CreatedBy: "steward"}
	first, err := r.Resolve(base)
	if err != nil || first.Revision != 1 {
		t.Fatal(first, err)
	}
	base.Config["instructions"] = "changed"
	base.CreatedBy = "human"
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.Save(base, 1)
			if err == nil {
				wins.Add(1)
			} else if !errors.Is(err, revision.ErrConflict) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatal(wins.Load())
	}
	v, err := r.Resolve(first)
	if err != nil || v.Revision != 2 || v.CreatedBy != "human" {
		t.Fatal(v, err)
	}
	r.Deregister(base.Name)
	h, err := r.History(base.Name)
	if err != nil || len(h) != 3 || h[0].Config["instructions"] != "original" || !h[2].Retired {
		t.Fatal(h, err)
	}
	if _, err = r.Resolve(first); err == nil {
		t.Fatal("retired agent was relaunched")
	}
}
