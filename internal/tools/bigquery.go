package tools

import (
	"context"
	"fmt"

	"github.com/idristarwala/agent-factory/internal/secrets"
)

// BigQueryClient is the minimal interface we need.
type BigQueryClient interface {
	Query(ctx context.Context, sql string) (string, error)
}

// BigQueryAuth is one of the supported authentication methods.
// BigQuery supports several; we expose all of them.
type BigQueryAuth string

const (
	// BQAuthADC uses Application Default Credentials
	// (gcloud auth application-default, GCE/GKE metadata, etc.).
	BQAuthADC BigQueryAuth = "adc"
	// BQAuthServiceAccount uses a JSON key file from the secrets dir.
	BQAuthServiceAccount BigQueryAuth = "service_account"
	// BQAuthWorkloadIdentity uses Workload Identity Federation
	// (external account JSON from secrets dir).
	BQAuthWorkloadIdentity BigQueryAuth = "workload_identity"
	// BQAuthImpersonation uses a service account that impersonates another.
	BQAuthImpersonation BigQueryAuth = "impersonation"
)

// RealBigQueryTool supports every common BigQuery auth path.
type RealBigQueryTool struct {
	Project string
	Auth    BigQueryAuth
	Client  BigQueryClient
	Secrets *secrets.Loader
	// Impersonate is the target service account email when Auth == impersonation.
	Impersonate string
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
	if t.Client == nil {
		return "", fmt.Errorf("bigquery client not configured")
	}
	switch t.Auth {
	case BQAuthServiceAccount:
		if t.Secrets == nil || !t.Secrets.Exists("bigquery-sa.json") {
			return "", fmt.Errorf("bigquery-sa.json required for service_account auth")
		}
	case BQAuthWorkloadIdentity:
		if t.Secrets == nil || !t.Secrets.Exists("bigquery-wif.json") {
			return "", fmt.Errorf("bigquery-wif.json required for workload_identity auth")
		}
	case BQAuthImpersonation:
		if t.Impersonate == "" {
			return "", fmt.Errorf("Impersonate email required")
		}
	}
	q, _ := args["query"].(string)
	return t.Client.Query(ctx, q)
}

// CredentialPath returns the secrets-dir path for the chosen auth method.
func (t *RealBigQueryTool) CredentialPath() string {
	if t.Secrets == nil {
		return ""
	}
	switch t.Auth {
	case BQAuthServiceAccount:
		return t.Secrets.Path("bigquery-sa.json")
	case BQAuthWorkloadIdentity:
		return t.Secrets.Path("bigquery-wif.json")
	default:
		return ""
	}
}
