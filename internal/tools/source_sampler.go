package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var ErrSourceDenied = errors.New("source unavailable for this factory")
var ErrSourceParameter = errors.New("unsupported source query parameter")

// SourceSampler uses operator-configured endpoints and credentials. Callers may
// select declared query parameters but cannot substitute the URL or key file.
type SourceSampler struct{ config *EvidenceConfig }

func LoadSourceSampler(path string) (*SourceSampler, error) {
	config, err := loadHTTPConfig(path)
	if err != nil {
		return nil, err
	}
	for name, tool := range config.Tools {
		if strings.ContainsAny(tool.URLTemplate, "{}") {
			return nil, fmt.Errorf("source %q must have a fixed sampling URL", name)
		}
		tool.CacheTTL = 0
	}
	return &SourceSampler{config: config}, nil
}
func (s *SourceSampler) CheckCredentials() error { return s.config.CheckCredentials() }

func (s *SourceSampler) Sample(ctx context.Context, factory, name string, query map[string]string) (json.RawMessage, error) {
	tool := s.config.Tools[name]
	if tool == nil || factory == "" || !CheckGrant(s.config.Grants, factory, name, "read-only") {
		return nil, ErrSourceDenied
	}
	for k := range query {
		if !slices.Contains(tool.QueryParams, k) {
			return nil, ErrSourceParameter
		}
	}
	raw, err := tool.Call(ctx, query)
	if err != nil {
		return nil, err
	}
	if !json.Valid([]byte(raw)) {
		return nil, fmt.Errorf("source returned invalid JSON")
	}
	return json.RawMessage(raw), nil
}
