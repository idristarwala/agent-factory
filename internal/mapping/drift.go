package mapping

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
)

// PathShape stores observed JSON kinds, including "missing" separately from "null".
type PathShape struct {
	Path  string   `json:"path"`
	Kinds []string `json:"kinds"`
}

func observedKind(row any, path []string) string {
	v, ok := lookup(row, path)
	if !ok {
		return "missing"
	}
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case json.Number:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}

func (m SourceMapping) Observe(sample any) ([]PathShape, error) {
	records, err := m.records(sample)
	if err != nil {
		return nil, err
	}
	paths := slices.Clone(m.Identity)
	for _, p := range m.Fields {
		paths = append(paths, p)
	}
	relationPaths := map[string]bool{}
	for _, rel := range m.Relationships {
		relationPaths[rel.Path] = true
		paths = append(paths, rel.Path)
	}
	sort.Strings(paths)
	paths = slices.Compact(paths)
	shapes := make([]PathShape, 0, len(paths))
	for _, p := range paths {
		parts, err := tokens(p)
		if err != nil {
			return nil, err
		}
		var kinds []string
		for _, row := range records {
			kind := observedKind(row, parts)
			if !slices.Contains(kinds, kind) {
				kinds = append(kinds, kind)
			}
		}
		if relationPaths[p] && len(kinds) == 1 && kinds[0] == "missing" {
			return nil, fmt.Errorf("relationship path %s was absent from the approval sample", p)
		}
		sort.Strings(kinds)
		shapes = append(shapes, PathShape{p, kinds})
	}
	return shapes, nil
}

type ApprovedMapping struct {
	SourceMapping
	Observed         []PathShape `json:"observed,omitempty"`
	ProposalRevision *int        `json:"proposal_revision,omitempty"`
}

func (m ApprovedMapping) ImportRows(payload any) ([]PreviewRow, error) {
	rows, err := m.Preview(payload)
	if err != nil || len(m.Observed) == 0 {
		return rows, err
	}
	records, err := m.records(payload)
	if err != nil {
		return nil, err
	}
	for _, shape := range m.Observed {
		parts, err := tokens(shape.Path)
		if err != nil {
			return nil, err
		}
		for i, row := range records {
			if rows[i].Problem != "" {
				continue
			}
			kind := observedKind(row, parts)
			if !slices.Contains(shape.Kinds, kind) {
				rows[i].Problem = fmt.Sprintf("source shape changed at %s: observed %s, approved %v", shape.Path, kind, shape.Kinds)
				rows[i].Record = nil
				rows[i].Location = nil
			}
		}
	}
	return rows, nil
}
