package tools

import (
	"context"
	"fmt"
)

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
