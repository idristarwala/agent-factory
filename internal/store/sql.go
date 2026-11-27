package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
	"os"
	"strconv"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
	"modernc.org/sqlite"
)

// SQLStore is a database/sql implementation.
// Works with SQLite (default), Postgres, or any driver that supports the schema.
// Domain-agnostic.
type SQLStore struct {
	db *sql.DB
}

// busyConnector applies the workspace timeout after the driver's DSN setup on
// every physical connection, including replacements opened by database/sql.
type busyConnector struct {
	driver.Connector
	pragma string
}

func (c busyConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	// modernc's SQLite connections implement driver.ExecerContext.
	if _, err := conn.(driver.ExecerContext).ExecContext(ctx, c.pragma, nil); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func NewSQLStore(dsn string) (*SQLStore, error) {
	timeout := int64(5000)
	if raw, ok := os.LookupEnv("FACTORY_SQLITE_BUSY_TIMEOUT_MS"); ok {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("FACTORY_SQLITE_BUSY_TIMEOUT_MS must be an integer from 0 to 2147483647")
		}
		timeout = n
	}
	base, err := sqlite.NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(busyConnector{Connector: base, pragma: fmt.Sprintf("PRAGMA busy_timeout=%d", timeout)})
	db.SetMaxOpenConns(1)
	s := &SQLStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLStore) migrate() error {
	return migrateSchema(s.db, sqliteEntryID)
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
