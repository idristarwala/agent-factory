package mapping

import (
	"context"
	"fmt"
)

// BigQuerySource pulls rows from BigQuery.
type BigQuerySource struct {
	Project string
	Dataset string
	QueryFn func(ctx context.Context, sql string, limit int) ([]Row, error)
}

func (s *BigQuerySource) Name() string { return "bigquery:" + s.Project + "." + s.Dataset }

func (s *BigQuerySource) Fetch(ctx context.Context, query string, limit int) ([]Row, error) {
	if s.QueryFn == nil {
		return nil, fmt.Errorf("bigquery client not configured")
	}
	return s.QueryFn(ctx, query, limit)
}

// APISource pulls rows from a REST API.
type APISource struct {
	Name_   string
	BaseURL string
	FetchFn func(ctx context.Context, query string, limit int) ([]Row, error)
}

func (s *APISource) Name() string { return s.Name_ }

func (s *APISource) Fetch(ctx context.Context, query string, limit int) ([]Row, error) {
	if s.FetchFn == nil {
		return nil, fmt.Errorf("api client not configured")
	}
	return s.FetchFn(ctx, query, limit)
}
