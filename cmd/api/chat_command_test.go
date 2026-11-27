package main

import (
	"testing"

	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
)

func TestChatFailedStartDoesNotConfirmOrPersistRun(t *testing.T) {
	ws := &factory.Workspace{Store: revision.NewMemory()}
	source := &factory.RunRequest{Factory: "missing", Records: []schema.Entity{{ID: "r1", Type: "record"}}}
	reply, handled, err := chatCommand(ws, "Could you start a run to update the record?", source)
	if !handled || err == nil || reply != "" {
		t.Fatalf("reply %q, handled %v, error %v", reply, handled, err)
	}
	runs, err := ws.List()
	if err != nil || len(runs) != 0 {
		t.Fatalf("unexpected runs %v, error %v", runs, err)
	}
}
