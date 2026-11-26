package tools

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SQLTool runs read-only queries against any database/sql driver
// (Postgres, MySQL, SQLite, etc.).
type SQLTool struct {
	name string
	db   *sql.DB
}

func NewSQLTool(name string, db *sql.DB) *SQLTool {
	return &SQLTool{name: name, db: db}
}

func (t *SQLTool) Name() string        { return t.name }
func (t *SQLTool) Description() string { return "Run a read-only SQL query. Returns up to 100 rows as JSON." }
func (t *SQLTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "SELECT statement"},
		},
		"required": []string{"query"},
	}
}

func (t *SQLTool) Call(ctx context.Context, args map[string]any) (string, error) {
	q, _ := args["query"].(string)
	q = strings.TrimSpace(q)
	if !strings.HasPrefix(strings.ToUpper(q), "SELECT") {
		return "", fmt.Errorf("only SELECT allowed")
	}
	rows, err := t.db.QueryContext(ctx, q)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var results []map[string]any
	for rows.Next() && len(results) < 100 {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		row := map[string]any{}
		for i, c := range cols {
			row[c] = vals[i]
		}
		results = append(results, row)
	}
	return fmt.Sprintf("%v", results), nil
}

// BigQueryTool is a placeholder for Google BigQuery.
// Real implementation would use cloud.google.com/go/bigquery.
type BigQueryTool struct {
	Project string
	Dataset string
}

func (t *BigQueryTool) Name() string        { return "bigquery" }
func (t *BigQueryTool) Description() string { return "Run a BigQuery SQL query (read-only)." }
func (t *BigQueryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
		"required": []string{"query"},
	}
}

func (t *BigQueryTool) Call(ctx context.Context, args map[string]any) (string, error) {
	// Stub: real impl uses bigquery.Client.Query
	return fmt.Sprintf("[bigquery stub] would run: %v", args["query"]), nil
}
