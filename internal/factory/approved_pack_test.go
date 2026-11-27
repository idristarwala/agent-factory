package factory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/pack"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type packPolicyGateway struct{ roles []string }

func (g *packPolicyGateway) Generate(req models.Request) (*models.Response, error) {
	if !strings.Contains(req.System+req.User, "Keep the source identifier") || strings.Contains(req.System+req.User, "Disabled policy") {
		return nil, fmt.Errorf("unexpected policies for %s", req.CallerRole)
	}
	g.roles = append(g.roles, req.CallerRole)
	return scripted{}.Generate(req)
}

func TestWorkspaceExecutesPinnedReviewedPack(t *testing.T) {
	st := revision.NewMemory()
	raw := []byte("name: support\nnode_types: [request]\nworker_roles: [worker]\nrules:\n - {id: retain, text: Keep the source identifier}\n - {id: disabled, text: Disabled policy, active: false}")
	sum := fmt.Sprintf("%x", sha256.Sum256(raw))
	proof := []byte(fmt.Sprintf(`{"status":"passed","commit":"test-commit","pack_sha256":"%s","tests":["fixture"]}`, sum))
	release := pack.Release{Status: "proposed", Commit: "test-commit", Pack: raw, Evidence: proof, PackSHA: sum, EvidenceSHA: fmt.Sprintf("%x", sha256.Sum256(proof))}
	save := func(key string, expected int, value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := revision.SaveDocument(st, key, expected, "human", data); err != nil {
			t.Fatal(err)
		}
	}
	save("pack-release/support", 0, release)
	ref := &PackRef{Name: "support", Version: 1}
	spec := Spec{RecordType: "request", Worker: "worker", Fields: map[string]string{"priority": "integer"}, PackRelease: ref}
	if _, err := pinApprovedPack(st, spec); err == nil {
		t.Fatal("accepted proposed release")
	}
	release.Status = "active"
	release.ProposalRevision = 1
	save("pack-release/support", 1, release)
	ref.Version = 2
	spec.ProposeTool = "ungranted"
	if _, err := pinApprovedPack(st, spec); err == nil {
		t.Fatal("accepted tool outside pack")
	}
	spec.ProposeTool = ""
	save("factory/demo", 0, spec)
	gw := &packPolicyGateway{}
	ws := Workspace{Store: st, Gateway: gw}
	run, err := ws.Start(RunRequest{Factory: "demo", Description: "Set priority", SelectedRules: map[string]int{}, Records: []schema.Entity{{ID: "a", Type: "request", Attributes: map[string]any{"priority": 1}}}})
	if err != nil {
		t.Fatal(err)
	}
	release.Status = "rejected"
	save("pack-release/support", 2, release)
	if err := ws.Execute(run.ID); err != nil {
		t.Fatal(err)
	}
	saved, err := ws.Get(run.ID)
	if err != nil || saved.Plan.ApprovedPack.Version != 2 || saved.Plan.ApprovedPack.PackSHA != sum || strings.Join(gw.roles, ",") != "worker,judge" || saved.Status != "in_review" {
		t.Fatal(saved.Error, gw.roles, err)
	}
	ref.Version = 3
	if _, err := pinApprovedPack(st, spec); err == nil {
		t.Fatal("accepted rejected release")
	}
}
