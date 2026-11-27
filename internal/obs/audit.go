package obs

import (
	"encoding/json"
	"sync"
	"time"
)

type AuditEntry struct {
	Feedback  string          `json:"feedback,omitempty"`
	Proposal  json.RawMessage `json:"proposal,omitempty"`
	TraceID   string          `json:"trace_id"`
	EntityID  string          `json:"entity_id"`
	Action    string          `json:"action"`
	Before    map[string]any  `json:"before,omitempty"`
	After     map[string]any  `json:"after,omitempty"`
	Reason    string          `json:"reason"`
	Actor     string          `json:"actor"`
	Timestamp time.Time       `json:"timestamp"`
}

type AuditLog struct {
	mu      sync.Mutex
	entries []AuditEntry
}

func NewAuditLog() *AuditLog { return &AuditLog{} }

func (a *AuditLog) Append(e AuditEntry) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	a.mu.Lock()
	a.entries = append(a.entries, e)
	a.mu.Unlock()
}

func (a *AuditLog) All() []AuditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]AuditEntry, len(a.entries))
	copy(out, a.entries)
	return out
}
