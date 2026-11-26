package agents

import (
	"fmt"
	"sync"
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
}

// Registry tracks which agent versions are currently running.
type Registry struct {
	mu     sync.RWMutex
	active map[string]AgentVersion // name -> version
}

func NewRegistry() *Registry {
	return &Registry{active: make(map[string]AgentVersion)}
}

func (r *Registry) Register(v AgentVersion) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.StartedAt.IsZero() {
		v.StartedAt = time.Now()
	}
	r.active[v.Name] = v
}

func (r *Registry) Deregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.active, name)
}

func (r *Registry) Running() []AgentVersion {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AgentVersion, 0, len(r.active))
	for _, v := range r.active {
		out = append(out, v)
	}
	return out
}

func (r *Registry) Get(name string) (AgentVersion, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.active[name]
	return v, ok
}

func (r *Registry) Summary() string {
	running := r.Running()
	if len(running) == 0 {
		return "no agents running"
	}
	s := fmt.Sprintf("%d agents running:\n", len(running))
	for _, v := range running {
		s += fmt.Sprintf("- %s (%s) v%s tier=%s since %s\n",
			v.Name, v.Role, v.Version, v.ModelTier, v.StartedAt.Format(time.RFC3339))
	}
	return s
}
