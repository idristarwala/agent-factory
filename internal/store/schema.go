package store

import (
	"database/sql"
	"fmt"
)

const (
	sqliteEntryID   = "INTEGER PRIMARY KEY AUTOINCREMENT"
	postgresEntryID = "SERIAL PRIMARY KEY"
)

func migrateSchema(db interface {
	Exec(string, ...any) (sql.Result, error)
}, entryIDType string) error {
	_, err := db.Exec(fmt.Sprintf(`
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
	CREATE TABLE IF NOT EXISTS unit_payloads (id TEXT PRIMARY KEY REFERENCES units(id), data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS run_versions (id TEXT, version INTEGER, data TEXT, PRIMARY KEY(id,version));
 CREATE TABLE IF NOT EXISTS memory_file_versions (id TEXT, version INTEGER, data TEXT, PRIMARY KEY(id,version));
 CREATE TABLE IF NOT EXISTS config_versions (id TEXT, version INTEGER, data TEXT, PRIMARY KEY(id,version));
 CREATE TABLE IF NOT EXISTS agent_versions (id TEXT, version INTEGER, data TEXT, PRIMARY KEY(id,version));
 CREATE TABLE IF NOT EXISTS rule_versions (
		id TEXT,
		version INTEGER,
		data TEXT,
		PRIMARY KEY (id, version)
	);
	CREATE TABLE IF NOT EXISTS entries (
		id %s,
		kind TEXT,
		content TEXT,
		source TEXT,
		meta TEXT
	);
	`, entryIDType))
	return err
}
