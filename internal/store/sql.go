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
	return migrateSchema(s.db, "INTEGER PRIMARY KEY AUTOINCREMENT")
}

func (s *SQLStore) GetNode(id graph.NodeID) (*graph.Node, error) { return readNode(s.db, id) }
func (s *SQLStore) UpsertNode(n graph.Node) error                { return writeNode(s.db, n) }
func (s *SQLStore) ProposeMutation(m *graph.Mutation) error      { return writeMutation(s.db, m) }

func (s *SQLStore) CreateUnit(u *unitofwork.UnitOfWork) error         { return createUnit(s.db, u) }
func (s *SQLStore) GetUnit(id string) (*unitofwork.UnitOfWork, error) { return readUnit(s.db, id) }
func (s *SQLStore) UpdateUnitStatus(id string, status unitofwork.Status) error {
	return unitUpdate(s.db.Exec("UPDATE units SET status=$1 WHERE id=$2", status, id))
}
func (s *SQLStore) AttachProposal(id string, m *graph.Mutation) error {
	return attachProposal(s.db, id, m)
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
