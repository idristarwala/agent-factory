package memory

import (
	"sync"
	"time"
)

// Store holds shared memory and extracted learnings.
// Agents read from and write to it; learnings can later become rules.

type Entry struct {
	ID        string
	Kind      string // "memory" or "learning"
	Content   string
	Source    string
	CreatedAt time.Time
	Metadata  map[string]any
}

type Store struct {
	mu      sync.RWMutex
	entries []Entry
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Add(kind, content, source string, meta map[string]any) Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := Entry{
		ID:        time.Now().Format("20060102150405.000"),
		Kind:      kind,
		Content:   content,
		Source:    source,
		CreatedAt: time.Now(),
		Metadata:  meta,
	}
	s.entries = append(s.entries, e)
	return e
}

func (s *Store) ByKind(kind string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Entry
	for _, e := range s.entries {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

func (s *Store) All() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}
