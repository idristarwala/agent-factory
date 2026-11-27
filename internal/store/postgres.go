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
	s, err := NewPostgresStoreFromDB(db)
	if err != nil {
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
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize concurrent replica startup, including initialization of an empty schema.
	if _, err = tx.Exec("SELECT pg_advisory_xact_lock(17012024, 1)"); err != nil {
		return err
	}
	if err = migrateSchema(tx, postgresEntryID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) GetNode(id graph.NodeID) (*graph.Node, error) { return readNode(s.db, id) }
func (s *PostgresStore) UpsertNode(n graph.Node) error                { return writeNode(s.db, n) }
func (s *PostgresStore) ProposeMutation(m *graph.Mutation) error      { return writeMutation(s.db, m) }

func (s *PostgresStore) CreateUnit(u *unitofwork.UnitOfWork) error         { return createUnit(s.db, u) }
func (s *PostgresStore) GetUnit(id string) (*unitofwork.UnitOfWork, error) { return readUnit(s.db, id) }
func (s *PostgresStore) UpdateUnitStatus(id string, status unitofwork.Status) error {
	return unitUpdate(s.db.Exec("UPDATE units SET status=$1 WHERE id=$2", status, id))
}
func (s *PostgresStore) AttachProposal(id string, m *graph.Mutation) error {
	return attachProposal(s.db, id, m)
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
