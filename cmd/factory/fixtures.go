package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/idristarwala/agent-factory/internal/factory"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/pack"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type packFixture struct {
	Definition string          `json:"definition"`
	Goal       string          `json:"goal"`
	Record     schema.Entity   `json:"record"`
	Worker     json.RawMessage `json:"worker_reply"`
	Judge      string          `json:"judge_reply"`
	Expected   map[string]any  `json:"expected_patch"`
	Approved   *bool           `json:"expected_approved"`
}

func (f packFixture) Generate(req models.Request) (*models.Response, error) {
	switch req.CallerRole {
	case "worker":
		return &models.Response{Content: string(f.Worker), Model: "fixture-worker"}, nil
	case "judge":
		return &models.Response{Content: f.Judge, Model: "fixture-judge"}, nil
	default:
		return nil, fmt.Errorf("unexpected fixture caller %q", req.CallerRole)
	}
}

func runFixtures(paths []string) error {
	return eachPack(paths, func(path string, p *pack.Pack) error {
		dir := filepath.Dir(path)
		entries, err := os.ReadDir(filepath.Join(dir, "fixtures"))
		if err != nil {
			return err
		}
		count := 0
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			file := filepath.Join(dir, "fixtures", entry.Name())
			if err := checkFixture(file, dir, p); err != nil {
				return fmt.Errorf("%s: %w", file, err)
			}
			fmt.Println("PASS", file)
			count++
		}
		if count == 0 {
			return fmt.Errorf("%s: no JSON fixtures", path)
		}
		return nil
	})
}

func checkFixture(path, dir string, p *pack.Pack) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	var f packFixture
	if err := dec.Decode(&f); err != nil {
		return err
	}
	if dec.Decode(new(any)) != io.EOF {
		return fmt.Errorf("expected one JSON fixture")
	}
	if f.Definition == "" || len(f.Worker) == 0 || f.Judge == "" || f.Approved == nil || f.Expected == nil {
		return fmt.Errorf("supply a definition, both gateway replies, expected_patch and expected_approved")
	}
	def, err := os.ReadFile(filepath.Join(dir, f.Definition))
	if err != nil {
		return err
	}
	var spec factory.Spec
	if err := json.Unmarshal(def, &spec); err != nil {
		return err
	}
	worker := spec.Worker
	if worker == "" {
		worker = "worker"
	}
	if !slices.Contains(p.WorkerRoles, worker) {
		return fmt.Errorf("worker %q is absent from pack worker_roles", worker)
	}
	if len(p.NodeTypes) > 0 && !slices.Contains(p.NodeTypes, spec.RecordType) {
		return fmt.Errorf("record type %q is absent from pack node_types", spec.RecordType)
	}
	st := revision.NewMemory()
	if _, err := revision.SaveDocument(st, "factory/"+p.Name, 0, "fixture", def); err != nil {
		return err
	}
	engine := rules.NewStoredEngine(st)
	for _, r := range p.Rules {
		if _, err := engine.Save(rules.VersionedRule{ID: r.ID, Description: r.Text, Active: r.Active, CreatedBy: "fixture"}, 0); err != nil {
			return err
		}
	}
	ws := factory.Workspace{Store: st, Gateway: f}
	run, err := ws.Start(factory.RunRequest{Description: f.Goal, Factory: p.Name, Records: []schema.Entity{f.Record}})
	if err != nil {
		return err
	}
	if err := ws.Execute(run.ID); err != nil {
		return err
	}
	run, err = ws.Get(run.ID)
	if err != nil {
		return err
	}
	if len(run.Decisions) != 1 {
		return fmt.Errorf("run %s: %s", run.Status, run.Error)
	}
	review := run.Decisions[0]
	actual, err := json.Marshal(review.After)
	if err != nil {
		return err
	}
	expected, err := json.Marshal(f.Expected)
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, expected) || (review.Status == "pending") != *f.Approved {
		return fmt.Errorf("got patch %s and judge status %q; expected patch %s and approved=%t", actual, review.Status, expected, *f.Approved)
	}
	return nil
}
