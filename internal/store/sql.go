package store

import (
	"database/sql"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
	_ "modernc.org/sqlite" // pure Go SQLite
)

// SQLStore is a database/sql implementation.
// Works with SQLite (default), Postgres, or any driver that supports the schema.
// Domain-agnostic.
type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(dsn string) (*SQLStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, err
	}
	s := &SQLStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		type TEXT,
		props TEXT
	);
	CREATE TABLE IF NOT EXISTS units (
		id TEXT PRIMARY KEY,
		goal_id TEXT,
		description TEXT,
		status TEXT,
		proposal TEXT
	);
	CREATE TABLE IF NOT EXISTS run_versions (id TEXT, version INTEGER, data TEXT, PRIMARY KEY(id,version));
 CREATE TABLE IF NOT EXISTS config_versions (id TEXT, version INTEGER, data TEXT, PRIMARY KEY(id,version));
 CREATE TABLE IF NOT EXISTS agent_versions (id TEXT, version INTEGER, data TEXT, PRIMARY KEY(id,version));
 CREATE TABLE IF NOT EXISTS rule_versions (
		id TEXT,
		version INTEGER,
		data TEXT,
		PRIMARY KEY (id, version)
	);
	CREATE TABLE IF NOT EXISTS entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind TEXT,
		content TEXT,
		source TEXT,
		meta TEXT
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLStore) GetNode(id graph.NodeID) (*graph.Node, error) {
	var typ, props string
	err := s.db.QueryRow("SELECT type, props FROM nodes WHERE id = ?", id).Scan(&typ, &props)
	if err != nil {
		return nil, err
	}
	return &graph.Node{ID: id, Type: typ}, nil
}

func (s *SQLStore) UpsertNode(n graph.Node) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO nodes (id, type, props) VALUES (?, ?, ?)", n.ID, n.Type, "")
	return err
}

func (s *SQLStore) ProposeMutation(m *graph.Mutation) error {
	for _, n := range m.Updates {
		if err := s.UpsertNode(n); err != nil {
			return err
		}
	}
	for _, n := range m.Adds {
		if err := s.UpsertNode(n); err != nil {
			return err
		}
	}
	for _, id := range m.Deletes {
		_, err := s.db.Exec("DELETE FROM nodes WHERE id = ?", id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLStore) CreateUnit(u *unitofwork.UnitOfWork) error {
	_, err := s.db.Exec("INSERT INTO units (id, goal_id, description, status) VALUES (?, ?, ?, ?)",
		u.ID, u.GoalID, u.Description, string(u.Status))
	return err
}

func (s *SQLStore) GetUnit(id string) (*unitofwork.UnitOfWork, error) {
	var goal, desc, status string
	err := s.db.QueryRow("SELECT goal_id, description, status FROM units WHERE id = ?", id).Scan(&goal, &desc, &status)
	if err != nil {
		return nil, err
	}
	return &unitofwork.UnitOfWork{ID: id, GoalID: goal, Description: desc, Status: unitofwork.Status(status)}, nil
}

func (s *SQLStore) UpdateUnitStatus(id string, status unitofwork.Status) error {
	_, err := s.db.Exec("UPDATE units SET status = ? WHERE id = ?", string(status), id)
	return err
}

func (s *SQLStore) AttachProposal(id string, m *graph.Mutation) error {
	_, err := s.db.Exec("UPDATE units SET proposal = ? WHERE id = ?", "", id)
	return err
}

func (s *SQLStore) Definitions() revision.Store { return revision.SQL{DB: s.db} }
func (s *SQLStore) SaveRuleVersion(id string, v int, data []byte) error {
	return s.Definitions().Insert(revision.Rule, revision.Snapshot{ID: id, Version: v, Data: data})
}
func (s *SQLStore) ActiveRules() ([][]byte, error) { return latestRuleData(s.Definitions()) }
func (s *SQLStore) SaveEntry(kind, content, source string, meta []byte) error {
	_, err := s.db.Exec("INSERT INTO entries (kind, content, source, meta) VALUES (?, ?, ?, ?)", kind, content, source, string(meta))
	return err
}

func (s *SQLStore) EntriesByKind(kind string) ([][]byte, error) {
	rows, err := s.db.Query("SELECT content FROM entries WHERE kind = ?", kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][]byte
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, []byte(c))
	}
	return out, nil
}

func (s *SQLStore) Close() error {
	return s.db.Close()
}

func (s *SQLStore) String() string {
	return fmt.Sprintf("SQLStore")
}
