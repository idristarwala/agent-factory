package factory

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/schema"
	"github.com/idristarwala/agent-factory/internal/settings"
	"sort"
	"strings"
	"time"
)

var ErrNotFound = errors.New("run not found")

type RunRequest struct {
	Description string          `json:"description"`
	Factory     string          `json:"factory"`
	Revision    int             `json:"factory_revision"`
	Records     []schema.Entity `json:"records"`
	MaxTokens   int             `json:"max_tokens"`
}
type Call struct {
	Model  string  `json:"model"`
	Input  int     `json:"input_tokens"`
	Output int     `json:"output_tokens"`
	Total  int     `json:"tokens"`
	Cost   float64 `json:"estimated_usd"`
	Priced bool    `json:"priced"`
}
type Price struct {
	Input  float64 `json:"input_per_million"`
	Output float64 `json:"output_per_million"`
}
type Review struct {
	UpdatedAt time.Time       `json:"updated_at"`
	ID        string          `json:"id"`
	RecordID  string          `json:"entity_id"`
	Status    string          `json:"status"`
	Reason    string          `json:"reason"`
	Feedback  string          `json:"feedback,omitempty"`
	Proposal  *graph.Mutation `json:"proposal"`
	Before    map[string]any  `json:"before"`
	After     map[string]any  `json:"after"`
}
type Run struct {
	ID               string            `json:"id"`
	Version          int               `json:"version"`
	Description      string            `json:"description"`
	Status           string            `json:"status"`
	CreatedAt        time.Time         `json:"created_at"`
	Plan             Plan              `json:"plan"`
	Settings         settings.Settings `json:"settings"`
	SettingsRevision int               `json:"settings_revision"`
	Records          []schema.Entity   `json:"records"`
	Decisions        []Review          `json:"decisions"`
	Calls            []Call            `json:"calls"`
	Error            string            `json:"error,omitempty"`
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
	err = json.Unmarshal(h[len(h)-1].Data, &r)
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
		if err := json.Unmarshal(s.Data, &r); err != nil {
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
	if len(req.Records) > 0 || req.Factory != "" {
		if len(req.Records) == 0 {
			return Run{}, fmt.Errorf("supply source records")
		}
		plan, err := Prepare(w.Store, req.Factory, req.Revision)
		if err != nil {
			return Run{}, err
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
	if err != nil {
		return nil, err
	}
	c := Call{Model: resp.Model, Input: resp.Usage["prompt_tokens"], Output: resp.Usage["completion_tokens"], Total: resp.Usage["total_tokens"]}
	if c.Total == 0 {
		c.Total = c.Input + c.Output
	}
	if p, ok := m.prices[c.Model]; ok {
		c.Priced = true
		c.Cost = (float64(c.Input)*p.Input + float64(c.Output)*p.Output) / 1e6
	}
	m.calls = append(m.calls, c)
	m.remaining -= c.Total
	return resp, nil
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
	for i, e := range r.Records {
		mut, decision, evalErr := r.Plan.Evaluate(&meter, r.Description, e)
		// A cancellation wins even while a provider request is outstanding.
		latest, err := w.Get(id)
		if err != nil {
			return err
		}
		if latest.Status == "cancelled" {
			return nil
		}
		r.Calls = append(r.Calls, meter.calls...)
		meter.calls = nil
		if evalErr != nil {
			r.Status = "failed"
			r.Error = evalErr.Error()
			return w.write(&r)
		}
		d := newReview(r.ID, i, e, mut, decision)
		r.Decisions = append(r.Decisions, d)
		if i+1 == len(r.Records) {
			r.Status = reviewStatus(r.Decisions)
		}
		if err = w.write(&r); err != nil {
			return err
		}
	}
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
			return r, nil
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
		err = w.write(&r)
		return r, err
	}
	return Run{}, ErrNotFound
}
