package factory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/schema"
)

type reviewedCase struct {
	record schema.Entity
	review Review
}

type reviewedReplay struct {
	run        Run
	evidence   artifact.Evidence
	cases      []reviewedCase
	gateway    models.Gateway
	beforeCall func() error
}

// ReviewedReplay pins a human-reviewed run and evaluates proposed rules without applying mutations.
func (w *Workspace) ReviewedReplay(ref string, live graph.Graph, beforeCall func() error) (func(rules.VersionedRule, []rules.VersionedRule) (*rules.ReplayReport, error), error) {
	id, ver, ok := strings.Cut(ref, "@")
	version, err := strconv.Atoi(ver)
	if !ok || id == "" || err != nil || version < 1 {
		return nil, fmt.Errorf("expected a reviewed run reference RUN_ID@REVISION")
	}
	rows, err := w.Store.Read(revision.Run, id)
	if err != nil {
		return nil, err
	}
	var replay reviewedReplay
	found := false
	for _, row := range rows {
		if row.Version == version {
			if err := decodeRun(row.Data, &replay.run); err != nil {
				return nil, err
			}
			found = true
			break
		}
	}
	if !found {
		return nil, ErrNotFound
	}
	r := replay.run
	if r.Status != "completed" || len(r.Decisions) == 0 || len(r.Records) != len(r.Decisions) {
		return nil, fmt.Errorf("replay needs a completed workspace run with reviewed records")
	}
	snapshot, ok := live.Snapshot().(map[graph.NodeID]*graph.Node)
	if !ok {
		return nil, fmt.Errorf("cannot verify held-out record IDs")
	}
	for i, d := range r.Decisions {
		e := r.Records[i]
		if d.Status != "approved" || d.RecordID != e.ID || d.Before == nil || len(d.After) == 0 {
			return nil, fmt.Errorf("replay requires a human-approved expected patch for each record")
		}
		if _, overlap := snapshot[graph.NodeID(e.ID)]; overlap {
			return nil, fmt.Errorf("held-out record overlaps the live graph")
		}
		if len(e.ImageURLs) > 0 {
			return nil, fmt.Errorf("replay requires immutable image evidence; select a run without external image URLs")
		}
		e.Attributes = d.Before
		replay.cases = append(replay.cases, reviewedCase{e, d})
	}
	if r.Plan.UseGoalArtifacts {
		replay.evidence, err = artifact.Read(w.Store, r.Plan.Spec.Artifacts)
		if err != nil {
			return nil, err
		}
	}
	if w.Gateway == nil {
		return nil, fmt.Errorf("replay requires a gateway")
	}
	replay.beforeCall = beforeCall
	replay.gateway = w.Gateway
	return replay.compare, nil
}

func (r *reviewedReplay) compare(candidate rules.VersionedRule, baseline []rules.VersionedRule) (*rules.ReplayReport, error) {
	report := &rules.ReplayReport{Run: r.run.ID, Revision: r.run.Version, BaselineRules: map[string]int{}}
	for _, rule := range baseline {
		report.BaselineRules[rule.ID] = rule.Version
	}
	augmented := append(slices.Clone(baseline), candidate)
	for _, c := range r.cases {
		before, err := r.evaluate(c, baseline, candidate.ID, "baseline")
		if err != nil {
			return nil, err
		}
		after, err := r.evaluate(c, augmented, candidate.ID, "candidate")
		if err != nil {
			return nil, err
		}
		report.Cases = append(report.Cases, rules.ReplayCase{RecordID: c.record.ID, BaselinePassed: before, CandidatePassed: after})
	}
	return report, nil
}

type replayGateway struct {
	models.Gateway
	beforeCall func() error
	trace      string
	failure    error
}

func (g *replayGateway) Generate(req models.Request) (*models.Response, error) {
	if g.beforeCall != nil {
		if err := g.beforeCall(); err != nil {
			g.failure = err
			return nil, err
		}
	}
	req.UnitID = g.trace
	resp, err := g.Gateway.Generate(req)
	if err != nil {
		g.failure = err
	}
	return resp, err
}

func (r *reviewedReplay) evaluate(c reviewedCase, active []rules.VersionedRule, id, phase string) (bool, error) {
	p := r.run.Plan
	p.Rules = active
	gw := &replayGateway{Gateway: r.gateway, beforeCall: r.beforeCall, trace: fmt.Sprintf("replay/%s/%d/%s/%s/%s", r.run.ID, r.run.Version, id, phase, c.record.ID)}
	m, d, err := p.evaluate(gw, r.run.Description, c.record, c.review.Lineage, r.evidence, nil)
	if gw.failure != nil {
		return false, gw.failure
	}
	if errors.Is(err, agents.ErrInvalidPatch) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !d.Approved || m == nil || len(m.Updates) != 1 {
		return false, nil
	}
	actual, err := json.Marshal(m.Updates[0].Props)
	if err != nil {
		return false, err
	}
	expected, err := json.Marshal(c.review.After)
	if err != nil {
		return false, err
	}
	return bytes.Equal(actual, expected), nil
}
