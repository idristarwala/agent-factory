// Package revision stores immutable snapshots of factory definitions.
package revision

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
)

var ErrConflict = errors.New("definition has a newer revision")

type Kind string

const (
	Agent  Kind = "agent"
	Rule   Kind = "rule"
	Config Kind = "config"
	Run    Kind = "run"
)

type Snapshot struct {
	ID      string
	Version int
	Data    []byte
}
type Store interface {
	// Read returns revisions in ascending order. An empty ID lists the collection.
	Read(kind Kind, id string) ([]Snapshot, error)
	// Insert commits the next snapshot. Concurrent writers cannot replace it.
	Insert(kind Kind, s Snapshot) error
}
type Memory struct {
	mu      sync.RWMutex
	records map[Kind]map[string][]Snapshot
}

func NewMemory() *Memory { return &Memory{records: make(map[Kind]map[string][]Snapshot)} }
func (m *Memory) Read(k Kind, id string) ([]Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Snapshot
	for key, history := range m.records[k] {
		if id != "" && id != key {
			continue
		}
		for _, s := range history {
			s.Data = bytes.Clone(s.Data)
			result = append(result, s)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (m *Memory) Insert(k Kind, s Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.records[k] == nil {
		m.records[k] = make(map[string][]Snapshot)
	}
	if len(m.records[k][s.ID])+1 != s.Version {
		return ErrConflict
	}
	s.Data = bytes.Clone(s.Data)
	m.records[k][s.ID] = append(m.records[k][s.ID], s)
	return nil
}

type SQL struct{ DB *sql.DB }

func table(k Kind) (string, error) {
	switch k {
	case Agent:
		return "agent_versions", nil
	case Run:
		return "run_versions", nil
	case Config:
		return "config_versions", nil
	case Rule:
		return "rule_versions", nil
	}
	return "", fmt.Errorf("unknown definition kind %q", k)
}
func (db SQL) Read(k Kind, id string) ([]Snapshot, error) {
	t, err := table(k)
	if err != nil {
		return nil, err
	}
	query := "SELECT id,version,data FROM " + t
	var args []any
	if id != "" {
		query += " WHERE id=$1"
		args = append(args, id)
	}
	rows, err := db.DB.Query(query+" ORDER BY id, version", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Snapshot
	for rows.Next() {
		var s Snapshot
		if err := rows.Scan(&s.ID, &s.Version, &s.Data); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
func (db SQL) Insert(k Kind, s Snapshot) error {
	t, err := table(k)
	if err != nil {
		return err
	}
	// Both SQLite and PostgreSQL support numbered parameters and this conflict clause.
	r, err := db.DB.Exec("INSERT INTO "+t+" (id,version,data) SELECT $1,$2,$3 WHERE COALESCE((SELECT MAX(version) FROM "+t+" WHERE id=$1),0)+1=$2 ON CONFLICT (id,version) DO NOTHING", s.ID, s.Version, string(s.Data))
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err == nil && n == 0 {
		return ErrConflict
	}
	return err
}
