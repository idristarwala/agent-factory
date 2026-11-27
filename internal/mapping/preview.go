package mapping

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/idristarwala/agent-factory/internal/schema"
)

// SourceMapping describes how a record collection becomes inputs to a factory.
type SourceMapping struct {
	Collection string            `json:"collection"`
	RecordType string            `json:"record_type"`
	Identity   []string          `json:"identity"`
	Fields     map[string]string `json:"fields"` // destination attribute -> source pointer
}

type PreviewRow struct {
	Index   int            `json:"index"`
	Record  *schema.Entity `json:"record,omitempty"`
	Problem string         `json:"problem,omitempty"`
}

func tokens(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	if len(path) > 1024 || path[0] != '/' {
		return nil, fmt.Errorf("invalid JSON pointer")
	}
	parts := strings.Split(path[1:], "/")
	for i, p := range parts {
		for j := 0; j < len(p); j++ {
			if p[j] == '~' {
				j++
				if j == len(p) || (p[j] != '0' && p[j] != '1') {
					return nil, fmt.Errorf("invalid JSON pointer escape")
				}
			}
		}
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(p, "~1", "/"), "~0", "~")
	}
	return parts, nil
}

func lookup(value any, path []string) (any, bool) {
	for _, key := range path {
		switch v := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = v[key]
			if !ok {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(v) || strconv.Itoa(index) != key {
				return nil, false
			}
			value = v[index]
		default:
			return nil, false
		}
	}
	return value, true
}

func (m SourceMapping) Preview(sample any) ([]PreviewRow, error) {
	if strings.TrimSpace(m.RecordType) == "" || len(m.Identity) == 0 || len(m.Identity) > 16 || len(m.Fields) == 0 || len(m.Fields) > 128 {
		return nil, fmt.Errorf("supply a record type, 1–16 identity paths and 1–128 mapped fields")
	}
	var err error
	ids := make([][]string, len(m.Identity))
	seenPaths := map[string]bool{}
	for i, p := range m.Identity {
		if p == "" || seenPaths[p] {
			return nil, fmt.Errorf("identity paths must be distinct and nonempty")
		}
		seenPaths[p] = true
		ids[i], err = tokens(p)
		if err != nil {
			return nil, err
		}
	}
	fields := map[string][]string{}
	for name, p := range m.Fields {
		if strings.TrimSpace(name) == "" || p == "" {
			return nil, fmt.Errorf("mapped fields require a name and a nonempty path")
		}
		fields[name], err = tokens(p)
		if err != nil {
			return nil, err
		}
	}
	records, err := m.records(sample)
	if err != nil {
		return nil, err
	}
	out := make([]PreviewRow, len(records))
	seen := map[string]int{}
	size := 0
	for i, row := range records {
		out[i].Index = i
		if _, ok := row.(map[string]any); !ok {
			out[i].Problem = "record must be an object"
			continue
		}
		id, err := recordID(row, ids)
		if err != nil {
			out[i].Problem = err.Error()
			continue
		}
		if first, ok := seen[id]; ok {
			out[i].Problem = fmt.Sprintf("identity also used by row %d", first)
			out[first].Record = nil
			out[first].Problem = "duplicate identity"
			continue
		}
		seen[id] = i
		attrs := map[string]any{}
		for name, p := range fields {
			if value, ok := lookup(row, p); ok {
				attrs[name] = value
			}
		}
		record := schema.Entity{ID: id, Type: m.RecordType, Version: 1, Attributes: attrs}
		encoded, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		size += len(encoded)
		if size > 2<<20 {
			return nil, fmt.Errorf("preview exceeds 2 MiB")
		}
		out[i].Record = &record
	}
	return out, nil
}

func recordID(row any, paths [][]string) (string, error) {
	parts := make([]any, len(paths))
	for i, p := range paths {
		v, _ := lookup(row, p)
		switch x := v.(type) {
		case string:
			if strings.TrimSpace(x) == "" {
				return "", fmt.Errorf("identity component %d is empty", i+1)
			}
		case json.Number:
			if strings.ContainsAny(x.String(), ".eE") {
				return "", fmt.Errorf("identity component %d must be an integer", i+1)
			}
			if x == "-0" {
				v = json.Number("0")
			}
		default:
			return "", fmt.Errorf("identity component %d is missing or is not a string or integer", i+1)
		}
		parts[i] = v
	}
	if len(parts) == 1 {
		return fmt.Sprint(parts[0]), nil
	}
	b, err := json.Marshal(parts)
	return string(b), err
}

func (m SourceMapping) records(sample any) ([]any, error) {
	collection, err := tokens(m.Collection)
	if err != nil {
		return nil, err
	}
	selected, ok := lookup(sample, collection)
	if !ok {
		return nil, fmt.Errorf("collection path is missing")
	}
	records, ok := selected.([]any)
	if !ok {
		if _, ok := selected.(map[string]any); !ok {
			return nil, fmt.Errorf("collection must be an object or an array of objects")
		}
		records = []any{selected}
	}
	if len(records) == 0 || len(records) > 100 {
		return nil, fmt.Errorf("preview requires 1–100 records")
	}
	return records, nil
}

// SourceValue looks up a JSON pointer without converting numeric tokens.
func SourceValue(payload any, pointer string) (any, bool, error) {
	parts, err := tokens(pointer)
	if err != nil {
		return nil, false, err
	}
	value, ok := lookup(payload, parts)
	return value, ok, nil
}
