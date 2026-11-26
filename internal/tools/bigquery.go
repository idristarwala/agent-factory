package tools

import (
	"context"
	"fmt"
)

// BigQueryClient is the minimal interface we need.
// Real code uses cloud.google.com/go/bigquery.Client.
type BigQueryClient interface {
	Query(ctx context.Context, sql string) (string, error)
}

// RealBigQueryTool uses an injected client so tests don't need GCP creds.
type RealBigQueryTool struct {
	Project string
	Client  BigQueryClient
}

func (t *RealBigQueryTool) Name() string        { return "bigquery" }
func (t *RealBigQueryTool) Description() string { return "Run a BigQuery SQL query (read-only)." }
func (t *RealBigQueryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
		"required": []string{"query"},
	}
}

func (t *RealBigQueryTool) Call(ctx context.Context, args map[string]any) (string, error) {
	q, _ := args["query"].(string)
	if t.Client == nil {
		return "", fmt.Errorf("bigquery client not configured")
	}
	return t.Client.Query(ctx, q)
}
