package factory

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/memory"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/obs"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
	"github.com/idristarwala/agent-factory/internal/settings"
	"sort"
	"strings"
	"time"
)

var ErrNotFound = errors.New("run not found")

type RunRequest struct {
	UseGoalArtifacts bool            `json:"use_goal_artifacts,omitempty"`
	Import           *ImportPageRef  `json:"import,omitempty"`
	Memory           []memory.Ref    `json:"memory,omitempty"`
	Source           *SourceInput    `json:"source,omitempty"`
	SelectedRules    map[string]int  `json:"selected_rules"`
	Description      string          `json:"description"`
	Factory          string          `json:"factory"`
	Revision         int             `json:"factory_revision"`
	Records          []schema.Entity `json:"records"`
	MaxTokens        int             `json:"max_tokens"`
}
type Call struct {
	JudgeRoute string  `json:"judge_route,omitempty"`
	UnitID     string  `json:"unit_id,omitempty"`
	Failed     bool    `json:"failed,omitempty"`
	Role       string  `json:"role,omitempty"`
	Model      string  `json:"model"`
	Input      int     `json:"input_tokens"`
	Output     int     `json:"output_tokens"`
	Total      int     `json:"tokens"`
	Cost       float64 `json:"estimated_usd"`
	Priced     bool    `json:"priced"`
}
type Price struct {
	Input  float64 `json:"input_per_million"`
	Output float64 `json:"output_per_million"`
}
type Review struct {
	BrowserCapture string          `json:"browser_capture,omitempty"`
	Lineage        *SourceLineage  `json:"lineage,omitempty"`
	UpdatedAt      time.Time       `json:"updated_at"`
	ID             string          `json:"id"`
	RecordID       string          `json:"entity_id"`
	Status         string          `json:"status"`
	Reason         string          `json:"reason"`
	Feedback       string          `json:"feedback,omitempty"`
	Proposal       *graph.Mutation `json:"proposal"`
	Before         map[string]any  `json:"before"`
	After          map[string]any  `json:"after"`
}
type Run struct {
	Audit            []obs.AuditEntry         `json:"audit,omitempty"`
	Runner           string                   `json:"runner,omitempty"`
	Tools            map[string]int           `json:"tool_calls_by_name,omitempty"`
	Lineage          map[string]SourceLineage `json:"lineage,omitempty"`
	Import           *ImportPageRef           `json:"import,omitempty"`
	ImportProblems   []ImportProblem          `json:"import_problems,omitempty"`
	Source           *SourceRef               `json:"source,omitempty"`
	ID               string                   `json:"id"`
	Version          int                      `json:"version"`
	Description      string                   `json:"description"`
	Status           string                   `json:"status"`
	CreatedAt        time.Time                `json:"created_at"`
	Plan             Plan                     `json:"plan"`
	Settings         settings.Settings        `json:"settings"`
	SettingsRevision int                      `json:"settings_revision"`
	Records          []schema.Entity          `json:"records"`
	Decisions        []Review                 `json:"decisions"`
	Calls            []Call                   `json:"calls"`
	Error            string                   `json:"error,omitempty"`
}
type Workspace struct {
	Store   revision.Store
	Gateway models.Gateway
	Prices  map[string]Price
}

func (w *Workspace) write(r *Run) error {
	next := *r
	next.Version++
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err = w.Store.Insert(revision.Run, revision.Snapshot{ID: r.ID, Version: next.Version, Data: data}); err != nil {
		return err
	}
	*r = next
	return nil
}
func (w *Workspace) Get(id string) (Run, error) {
	if id == "" {
		return Run{}, ErrNotFound
	}
	h, err := w.Store.Read(revision.Run, id)
	if err != nil {
		return Run{}, err
	}
	if len(h) == 0 {
		return Run{}, ErrNotFound
	}
	var r Run
	err = decodeRun(h[len(h)-1].Data, &r)
	return r, err
}
func (w *Workspace) List() ([]Run, error) {
	h, err := w.Store.Read(revision.Run, "")
	if err != nil {
		return nil, err
	}
	out := []Run{}
	for i, s := range h {
		if i+1 < len(h) && h[i+1].ID == s.ID {
			continue
		}
		var r Run
		if err := decodeRun(s.Data, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
func (w *Workspace) Start(req RunRequest) (Run, error) {
	if strings.TrimSpace(req.Description) == "" || len(req.Records) > 100 || req.MaxTokens < 0 {
		return Run{}, fmt.Errorf("supply a goal, up to 100 source records and a nonnegative budget")
	}
	r := Run{ID: uuid.NewString(), Description: req.Description, Status: "created", CreatedAt: time.Now().UTC(), Records: req.Records}
	var editableFields *[]string
	if req.Import != nil {
		if req.Source != nil || len(req.Records) != 0 || req.Factory == "" || req.Import.ID == "" || req.Import.Page < 1 {
			return Run{}, fmt.Errorf("select one committed import page and its factory")
		}
		pages, err := w.ImportPages(req.Factory, req.Import.ID)
		if err != nil {
			return Run{}, err
		}
		if req.Import.Page > len(pages) {
			return Run{}, fmt.Errorf("unknown import page")
		}
		page := pages[req.Import.Page-1]
		req.Records = page.Records
		r.Records = page.Records
		r.Lineage = page.Lineage
		r.ImportProblems = page.Problems
		m, err := w.approvedMapping(page.Spec.SourceRef)
		if err != nil {
			return Run{}, err
		}
		editableFields = m.EditableFields
		r.Source = &page.Spec.SourceRef
		r.Import = req.Import
	}
	if req.Source != nil {
		if len(req.Records) != 0 || req.Factory == "" {
			return Run{}, fmt.Errorf("a mapped source requires a factory and cannot be combined with records")
		}
		batch, err := w.mappedRecords(req.Factory, *req.Source)
		if err != nil {
			return Run{}, err
		}
		editableFields = batch.editableFields
		req.Records = batch.records
		r.Records = batch.records
		r.ImportProblems = batch.problems
		r.Lineage = batch.lineage
		ref := req.Source.SourceRef
		r.Source = &ref
	}
	cfg, n, err := settings.Current(w.Store)
	if err != nil {
		return Run{}, err
	}
	r.Settings = cfg
	r.SettingsRevision = n
	if req.MaxTokens > 0 {
		if req.MaxTokens > cfg.MaxTokens {
			return Run{}, fmt.Errorf("budget exceeds the workspace limit")
		}
		r.Settings.MaxTokens = req.MaxTokens
	}
	if len(req.Records) > 0 || req.Factory != "" || len(req.Memory) > 0 {
		if len(req.Records) == 0 {
			return Run{}, fmt.Errorf("supply source records")
		}
		plan, err := Prepare(w.Store, req.Factory, req.Revision)
		if err != nil {
			return Run{}, err
		}
		if editableFields != nil {
			fields := map[string]string{}
			for _, key := range *editableFields {
				if typ, ok := plan.Spec.Fields[key]; ok {
					fields[key] = typ
				}
			}
			if len(fields) == 0 {
				return Run{}, fmt.Errorf("mapping and factory have no editable fields in common")
			}
			for _, key := range plan.Spec.Required {
				if _, ok := fields[key]; !ok {
					return Run{}, fmt.Errorf("mapping excludes required factory field %q", key)
				}
			}
			plan.Spec.Fields = fields
		}
		if req.SelectedRules != nil {
			selected := plan.Rules[:0]
			for _, rule := range plan.Rules {
				version, ok := req.SelectedRules[rule.ID]
				if !ok {
					continue
				}
				if version != rule.Version {
					return Run{}, fmt.Errorf("rule %s changed; reload available rules", rule.ID)
				}
				selected = append(selected, rule)
			}
			if len(selected) != len(req.SelectedRules) {
				return Run{}, fmt.Errorf("a selected rule is no longer active")
			}
			plan.Rules = selected
		}
		plan.Memory, err = (memory.Files{Store: w.Store, Factory: plan.Factory}).Pin(req.Memory)
		if err != nil {
			return Run{}, err
		}
		if req.UseGoalArtifacts {
			if len(plan.Spec.Artifacts) == 0 {
				return Run{}, fmt.Errorf("factory revision has no goal attachments")
			}
			if _, err := artifact.Read(w.Store, plan.Spec.Artifacts); err != nil {
				return Run{}, err
			}
			plan.UseGoalArtifacts = true
		}
		if plan.ApprovedPack != nil && plan.Spec.PackPromotion == nil {
			r.Settings.Observe = true
		}
		r.Plan = plan
		seen := map[string]bool{}
		for _, e := range r.Records {
			if e.ID == "" || e.Type != plan.Spec.RecordType || seen[e.ID] {
				return Run{}, fmt.Errorf("invalid or duplicate source record %q", e.ID)
			}
			seen[e.ID] = true
		}
		r.Status = "queued"
	}
	err = w.write(&r)
	return r, err
}

type measured struct {
	gateway   models.Gateway
	prices    map[string]Price
	remaining int
	deadline  time.Time
	calls     []Call
}

func (m *measured) Generate(req models.Request) (*models.Response, error) {
	if m.remaining <= 0 || time.Now().After(m.deadline) {
		return nil, errors.New("run budget exhausted")
	}
	if req.MaxTokens > m.remaining {
		req.MaxTokens = m.remaining
	}
	resp, err := m.gateway.Generate(req)
	c := modelCall(req, resp, err, m.prices)
	m.calls = append(m.calls, c)
	m.remaining -= c.Total
	return resp, err
}
func (w *Workspace) Execute(id string) error {
	r, err := w.Get(id)
	if err != nil {
		return err
	}
	if r.Status != "queued" {
		return revision.ErrConflict
	}
	r.Status = "running"
	if err = w.write(&r); err != nil {
		return err
	}
	meter := measured{gateway: w.Gateway, prices: w.Prices, remaining: r.Settings.MaxTokens, deadline: r.CreatedAt.Add(time.Duration(r.Settings.MaxMinutes) * time.Minute)}
	r.Tools = map[string]int{}
	var goalEvidence artifact.Evidence
	if r.Plan.UseGoalArtifacts {
		r.Tools["read_goal_artifacts"]++
		goalEvidence, err = artifact.Read(w.Store, r.Plan.Spec.Artifacts)
		if err != nil {
			r.Status = "failed"
			r.Error = err.Error()
			return w.checkpointEvaluation(&r)
		}
	}
	for i, e := range r.Records {
		var origin *SourceLineage
		if lineage, ok := r.Lineage[e.ID]; ok {
			origin = &lineage
		}
		recordEvidence := goalEvidence
		var capture BrowserCapture
		if tool := r.Plan.BrowserTool; tool != nil {
			r.Tools[tool.Name]++
			capture, err = tool.capture(w.Store, e, meter.deadline)
			if err != nil {
				r.Status = "failed"
				r.Error = err.Error()
				return w.checkpointEvaluation(&r)
			}
			recordEvidence = capture.evidence(recordEvidence)
		}
		mut, decision, evalErr := r.Plan.evaluate(&meter, r.Description, e, origin, recordEvidence, r.Tools)
		r.Calls = append(r.Calls, meter.calls...)
		meter.calls = nil
		if evalErr != nil {
			r.Status = "failed"
			r.Error = evalErr.Error()
			return w.checkpointEvaluation(&r)
		}
		d := newReview(r.ID, i, e, mut, decision)
		d.BrowserCapture = capture.ID
		d.Lineage = origin
		r.Decisions = append(r.Decisions, d)
		if i+1 == len(r.Records) {
			r.Status = reviewStatus(r.Decisions)
		}
		if err = w.checkpointEvaluation(&r); err != nil || r.Status == "cancelled" {
			return err
		}
	}
	return nil
}

// Cancellation wins the status race, while work already performed remains metered.
func (w *Workspace) checkpointEvaluation(r *Run) error {
	err := w.write(r)
	if !errors.Is(err, revision.ErrConflict) {
		return err
	}
	latest, err := w.Get(r.ID)
	if err != nil {
		return err
	}
	if latest.Status != "cancelled" {
		return revision.ErrConflict
	}
	latest.Calls = r.Calls
	latest.Tools = r.Tools
	if err = w.write(&latest); err != nil {
		return err
	}
	*r = latest
	return nil
}
func newReview(id string, index int, e schema.Entity, m *graph.Mutation, d agents.Decision) Review {
	status := "pending"
	if !d.Approved {
		status = "rejected"
	}
	return Review{UpdatedAt: time.Now().UTC(), ID: fmt.Sprintf("%s/%d", id, index), RecordID: e.ID, Status: status, Reason: d.Reason, Before: e.Attributes, After: m.Updates[0].Props, Proposal: m}
}
func reviewStatus(ds []Review) string {
	for _, d := range ds {
		if d.Status == "pending" {
			return "in_review"
		}
	}
	return "completed"
}
func (w *Workspace) Cancel(id string) (Run, error) {
	for {
		r, err := w.Get(id)
		if err != nil {
			return Run{}, err
		}
		switch r.Status {
		case "cancelled":
			return r, nil
		case "created", "queued", "running", "in_review":
		default:
			return Run{}, revision.ErrConflict
		}
		r.Status = "cancelled"
		if err = w.write(&r); errors.Is(err, revision.ErrConflict) {
			continue
		}
		return r, err
	}
}
func (w *Workspace) Decide(id, decisionID, status, note string, expected int) (Run, error) {
	r, err := w.Get(id)
	if err != nil {
		return Run{}, err
	}
	// An exact retry also lets the API finish persisting feedback after a transient error.
	for _, d := range r.Decisions {
		if d.ID == decisionID && (status == "approved" || status == "rejected") && d.Status == status && d.Feedback == note && expected < r.Version {
			return r, w.rememberReview(r, d)
		}
	}
	if r.Version != expected || r.Status != "in_review" {
		return Run{}, revision.ErrConflict
	}
	if status != "approved" && status != "rejected" {
		return Run{}, fmt.Errorf("invalid decision")
	}
	for i := range r.Decisions {
		d := &r.Decisions[i]
		if d.ID != decisionID {
			continue
		}
		if d.Status != "pending" {
			return Run{}, revision.ErrConflict
		}
		d.UpdatedAt = time.Now().UTC()
		d.Status = status
		d.Feedback = note
		if status == "approved" && !r.Settings.Observe {
			for j := range r.Records {
				e := &r.Records[j]
				if e.ID == d.RecordID {
					attrs := make(map[string]any, len(e.Attributes))
					for k, v := range e.Attributes {
						attrs[k] = v
					}
					for k, v := range d.After {
						attrs[k] = v
					}
					e.Attributes = attrs
					e.Version++
					break
				}
			}
		}
		r.Status = reviewStatus(r.Decisions)
		if err = w.write(&r); err != nil {
			return r, err
		}
		return r, w.rememberReview(r, *d)
	}
	return Run{}, ErrNotFound
}

// The durable decision remains the source of truth if a memory write must be retried.
func (w *Workspace) rememberReview(r Run, d Review) error {
	if d.Feedback == "" {
		return nil
	}
	content := d.Feedback
	if len(content) > 32<<10 {
		content = "Full reviewer feedback is attached to decision " + d.ID
	}
	files := memory.Files{Store: w.Store, Factory: r.Plan.Factory}
	path := "reviews/" + d.ID + ".txt"
	_, err := files.Write(path, content, "reviewer", 0)
	if errors.Is(err, revision.ErrConflict) {
		f, readErr := files.Read(memory.Ref{Path: path})
		if readErr != nil {
			return readErr
		}
		if f.Content == content {
			return nil
		}
	}
	return err
}
