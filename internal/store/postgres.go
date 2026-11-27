package store

import (
	"database/sql"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// PostgresStore is a durable Postgres implementation of Store.
// The caller must register a Postgres driver (e.g. import _ "github.com/lib/pq"
// or pgx) and pass an already-opened *sql.DB, or use NewPostgresStore.
//
// Schema uses Postgres types (TEXT, SERIAL, JSONB-compatible TEXT) and
// $n placeholders. Upserts use ON CONFLICT.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens a Postgres database with the given driver name
// (typically "postgres") and DSN, then runs migrations.
// The driver must already be registered via a blank import in the calling package.
func NewPostgresStore(driver, dsn string) (*PostgresStore, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &PostgresStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// NewPostgresStoreFromDB wraps an existing *sql.DB (useful for tests with
// a test container or shared pool). Runs migrations.
func NewPostgresStoreFromDB(db *sql.DB) (*PostgresStore, error) {
	s := &PostgresStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PostgresStore) migrate() error {
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
		id SERIAL PRIMARY KEY,
		kind TEXT,
		content TEXT,
		source TEXT,
		meta TEXT
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *PostgresStore) GetNode(id graph.NodeID) (*graph.Node, error) {
	var typ, props string
	err := s.db.QueryRow("SELECT type, props FROM nodes WHERE id = $1", id).Scan(&typ, &props)
	if err != nil {
		return nil, err
	}
	return &graph.Node{ID: id, Type: typ}, nil
}

func (s *PostgresStore) UpsertNode(n graph.Node) error {
	_, err := s.db.Exec(`
		INSERT INTO nodes (id, type, props) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET type = EXCLUDED.type, props = EXCLUDED.props
	`, n.ID, n.Type, "")
	return err
}

func (s *PostgresStore) ProposeMutation(m *graph.Mutation) error {
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
		if _, err := s.db.Exec("DELETE FROM nodes WHERE id = $1", id); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) CreateUnit(u *unitofwork.UnitOfWork) error {
	_, err := s.db.Exec(
		"INSERT INTO units (id, goal_id, description, status) VALUES ($1, $2, $3, $4)",
		u.ID, u.GoalID, u.Description, string(u.Status),
	)
	return err
}

func (s *PostgresStore) GetUnit(id string) (*unitofwork.UnitOfWork, error) {
	var goal, desc, status string
	err := s.db.QueryRow("SELECT goal_id, description, status FROM units WHERE id = $1", id).Scan(&goal, &desc, &status)
	if err != nil {
		return nil, err
	}
	return &unitofwork.UnitOfWork{ID: id, GoalID: goal, Description: desc, Status: unitofwork.Status(status)}, nil
}

func (s *PostgresStore) UpdateUnitStatus(id string, status unitofwork.Status) error {
	_, err := s.db.Exec("UPDATE units SET status = $1 WHERE id = $2", string(status), id)
	return err
}

func (s *PostgresStore) AttachProposal(id string, m *graph.Mutation) error {
	_, err := s.db.Exec("UPDATE units SET proposal = $1 WHERE id = $2", "", id)
	return err
}

func (s *PostgresStore) Definitions() revision.Store { return revision.SQL{DB: s.db} }
func (s *PostgresStore) SaveRuleVersion(id string, v int, data []byte) error {
	return s.Definitions().Insert(revision.Rule, revision.Snapshot{ID: id, Version: v, Data: data})
}
func (s *PostgresStore) ActiveRules() ([][]byte, error) { return latestRuleData(s.Definitions()) }
func (s *PostgresStore) SaveEntry(kind, content, source string, meta []byte) error {
	_, err := s.db.Exec(
		"INSERT INTO entries (kind, content, source, meta) VALUES ($1, $2, $3, $4)",
		kind, content, source, string(meta),
	)
	return err
}

func (s *PostgresStore) EntriesByKind(kind string) ([][]byte, error) {
	rows, err := s.db.Query("SELECT content FROM entries WHERE kind = $1", kind)
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

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
