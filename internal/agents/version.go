package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
	"maps"
	"slices"
	"time"
)

// AgentVersion identifies a running agent instance.
type AgentVersion struct {
	Name      string
	Role      string
	Version   string // semver or git sha
	ModelTier string
	StartedAt time.Time
	Config    map[string]string
	Tools     []string
	Revision  int
	CreatedBy string
	ChangedAt time.Time
	Retired   bool
}

// Registry tracks which agent versions are currently running.
type Registry struct{ store revision.Store }

func NewRegistry() *Registry                        { return NewStoredRegistry(revision.NewMemory()) }
func NewStoredRegistry(st revision.Store) *Registry { return &Registry{store: st} }
func (r *Registry) History(name string) ([]AgentVersion, error) {
	rows, err := r.store.Read(revision.Agent, name)
	if err != nil {
		return nil, err
	}
	result := make([]AgentVersion, 0, len(rows))
	for _, s := range rows {
		var v AgentVersion
		if err := json.Unmarshal(s.Data, &v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

// Save compares against the caller's observed head and returns the immutable definition.
func (r *Registry) Save(v AgentVersion, expected int) (AgentVersion, error) {
	h, err := r.History(v.Name)
	if err != nil {
		return AgentVersion{}, err
	}
	if len(h) != expected {
		return AgentVersion{}, revision.ErrConflict
	}
	if expected > 0 {
		last := h[len(h)-1]
		if v.Role == last.Role && v.Version == last.Version && v.ModelTier == last.ModelTier && v.Retired == last.Retired && maps.Equal(v.Config, last.Config) && slices.Equal(v.Tools, last.Tools) {
			return last, nil
		}
		v.StartedAt = last.StartedAt
	}
	v.ChangedAt = time.Now().UTC()
	if v.StartedAt.IsZero() {
		v.StartedAt = v.ChangedAt
	}
	if v.CreatedBy == "" {
		v.CreatedBy = "system"
	}
	v.Revision = expected + 1
	data, err := json.Marshal(v)
	if err != nil {
		return AgentVersion{}, err
	}
	if err = r.store.Insert(revision.Agent, revision.Snapshot{ID: v.Name, Version: v.Revision, Data: data}); err != nil {
		return AgentVersion{}, err
	}
	return v, nil
}
func (r *Registry) Register(v AgentVersion) AgentVersion {
	for {
		h, err := r.History(v.Name)
		if err != nil {
			panic(err)
		}
		out, err := r.Save(v, len(h))
		if errors.Is(err, revision.ErrConflict) {
			continue
		}
		if err != nil {
			panic(err)
		}
		return out
	}
}
func (r *Registry) Deregister(name string) {
	for {
		v, ok := r.Get(name)
		if !ok {
			return
		}
		v.Retired = true
		v.CreatedBy = "system"
		_, err := r.Save(v, v.Revision)
		if errors.Is(err, revision.ErrConflict) {
			continue
		}
		if err != nil {
			panic(err)
		}
		return
	}
}
func (r *Registry) Running() []AgentVersion {
	h, err := r.History("")
	if err != nil {
		panic(err)
	}
	out := []AgentVersion{}
	for i, v := range h {
		if (i+1 == len(h) || h[i+1].Name != v.Name) && !v.Retired {
			out = append(out, v)
		}
	}
	return out
}
func (r *Registry) Get(name string) (AgentVersion, bool) {
	h, err := r.History(name)
	if err != nil {
		panic(err)
	}
	if len(h) == 0 || h[len(h)-1].Retired {
		return AgentVersion{}, false
	}
	return h[len(h)-1], true
}

func (r *Registry) Summary() string {
	running := r.Running()
	if len(running) == 0 {
		return "no agents running"
	}
	s := fmt.Sprintf("%d agents running:\n", len(running))
	for _, v := range running {
		s += fmt.Sprintf("- %s (%s) rev=%d build=%s tier=%s since %s\n",
			v.Name, v.Role, v.Revision, v.Version, v.ModelTier, v.StartedAt.Format(time.RFC3339))
	}
	return s
}
