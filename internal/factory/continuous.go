package factory

import (
	"encoding/json"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/obs"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/settings"
)

func modelCall(req models.Request, resp *models.Response, err error, prices map[string]Price) Call {
	if resp == nil {
		return Call{JudgeRoute: req.JudgeRoute, UnitID: req.UnitID, Role: req.CallerRole, Failed: true}
	}
	c := Call{Failed: err != nil, JudgeRoute: req.JudgeRoute, UnitID: req.UnitID, Role: req.CallerRole, Model: resp.Model, Input: resp.Usage["prompt_tokens"], Output: resp.Usage["completion_tokens"], Total: resp.Usage["total_tokens"]}
	if c.Total == 0 {
		c.Total = c.Input + c.Output
	}
	if p, ok := prices[c.Model]; ok {
		c.Priced = true
		c.Cost = (float64(c.Input)*p.Input + float64(c.Output)*p.Output) / 1e6
	}
	return c
}

// ContinuousUsage records one invocation of the continuous runner, including concurrent workers.
type ContinuousUsage struct {
	MaxTokens int // Total reported token allowance. Zero means unlimited.
	reserved  int
	available *sync.Cond
	Gateway   models.Gateway
	Prices    map[string]Price
	mu        sync.Mutex
	tokens    int
	calls     []Call
	tools     map[string]int
}

func (u *ContinuousUsage) Generate(req models.Request) (*models.Response, error) {
	if u.MaxTokens > 0 {
		u.mu.Lock()
		if u.available == nil {
			u.available = sync.NewCond(&u.mu)
		}
		for u.tokens < u.MaxTokens && u.MaxTokens-u.tokens <= u.reserved {
			u.available.Wait()
		}
		if u.tokens >= u.MaxTokens {
			u.mu.Unlock()
			return nil, models.ErrTokenBudget
		}
		headroom := u.MaxTokens - u.tokens - u.reserved
		if req.MaxTokens <= 0 || req.MaxTokens > headroom {
			req.MaxTokens = headroom
		}
		u.reserved += req.MaxTokens
		u.mu.Unlock()
	}
	resp, err := u.Gateway.Generate(req)
	c := modelCall(req, resp, err, u.Prices)
	u.mu.Lock()
	u.tokens += c.Total
	u.calls = append(u.calls, c)
	if u.available != nil {
		u.reserved -= req.MaxTokens
		u.available.Broadcast()
	}
	u.mu.Unlock()
	return resp, err
}

func (u *ContinuousUsage) Tokens() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.tokens
}

func (u *ContinuousUsage) RecordTool(name string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.tools == nil {
		u.tools = make(map[string]int)
	}
	u.tools[name]++
}

// FinishContinuous publishes the terminal run to the same history read by the API.
func (w *Workspace) FinishContinuous(goal string, started time.Time, cfg settings.Settings, revision int, usage *ContinuousUsage, audit []obs.AuditEntry, runErr error) (Run, error) {
	usage.mu.Lock()
	r := Run{
		Audit: audit, Runner: "continuous", ID: uuid.NewString(), Description: goal, CreatedAt: started.UTC(),
		Settings: cfg, SettingsRevision: revision, Status: "completed",
		Calls: slices.Clone(usage.calls), Tools: maps.Clone(usage.tools),
	}
	usage.mu.Unlock()
	if runErr != nil {
		r.Status = "failed"
		r.Error = runErr.Error()
	}
	err := w.write(&r)
	return r, err
}

func ParsePrices(raw string) (map[string]Price, error) {
	p := map[string]Price{}
	if raw == "" {
		return p, nil
	}
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, err
	}
	for _, v := range p {
		if v.Input < 0 || v.Output < 0 {
			return nil, fmt.Errorf("negative model price")
		}
	}
	return p, nil
}
