package mapping

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type FieldProfile struct {
	Path           string         `json:"path"`
	Types          map[string]int `json:"types"`
	Missing        int            `json:"missing"`
	UniqueInSample bool           `json:"unique_in_sample"`
}
type CollectionProfile struct {
	CompositeKeys [][]string     `json:"composite_keys,omitempty"`
	Path          string         `json:"path"`
	Records       int            `json:"records"`
	Sampled       int            `json:"sampled"`
	Fields        []FieldProfile `json:"fields"`
}

func pointer(parent, name string) string {
	return parent + "/" + strings.NewReplacer("~", "~0", "/", "~1").Replace(name)
}

// ProfileSample discovers object collections in a JSON envelope. Child arrays in a
// record remain attributes until a mapping explicitly gives them their own identity.
func ProfileSample(sample any) ([]CollectionProfile, error) {
	var out []CollectionProfile
	visited := 0
	var visit func(any, string, int) error
	visit = func(v any, path string, depth int) error {
		visited++
		if visited > 8192 || len(path) > 1024 {
			return fmt.Errorf("sample envelope too large or complex")
		}
		if depth > 32 {
			return fmt.Errorf("payload nesting exceeds 32 levels")
		}
		switch x := v.(type) {
		case map[string]any:
			for k, child := range x {
				if err := visit(child, pointer(path, k), depth+1); err != nil {
					return err
				}
			}
		case []any:
			if len(x) == 0 {
				return nil
			}
			rows := make([]map[string]any, 0, min(len(x), 100))
			for _, item := range x {
				row, ok := item.(map[string]any)
				if !ok {
					return nil
				}
				if len(rows) < 100 {
					rows = append(rows, row)
				}
			}
			c, err := profileRows(path, len(x), rows)
			if err != nil {
				return err
			}
			if len(out) >= 32 {
				return fmt.Errorf("sample exceeds 32 candidate collections")
			}
			out = append(out, c)
		}
		return nil
	}
	if err := visit(sample, "", 0); err != nil {
		return nil, err
	}
	if obj, ok := sample.(map[string]any); ok {
		c, err := profileRows("", 1, []map[string]any{obj})
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("sample requires an object or a nonempty collection of objects")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func profileRows(path string, count int, rows []map[string]any) (CollectionProfile, error) {
	c := CollectionProfile{Path: path, Records: count, Sampled: len(rows), Fields: []FieldProfile{}}
	fields := map[string]*FieldProfile{}
	values := map[string]map[string]bool{}
	var walk func(map[string]any, string, int) error
	walk = func(obj map[string]any, parent string, depth int) error {
		if depth > 32 {
			return fmt.Errorf("record nesting exceeds 32 levels")
		}
		for k, v := range obj {
			p := pointer(parent, k)
			if len(p) > 1024 {
				return fmt.Errorf("field path exceeds 1024 bytes")
			}
			f := fields[p]
			if f == nil {
				if len(fields) >= 512 {
					return fmt.Errorf("sample collection exceeds 512 field paths")
				}
				f = &FieldProfile{Path: p, Types: map[string]int{}, Missing: len(rows)}
				fields[p] = f
			}
			f.Missing--
			kind := "null"
			var key string
			switch x := v.(type) {
			case map[string]any:
				kind = "object"
				if err := walk(x, p, depth+1); err != nil {
					return err
				}
			case []any:
				kind = "array"
			case bool:
				kind = "boolean"
			case string:
				kind = "string"
				if strings.TrimSpace(x) != "" {
					key = x
				}
			case json.Number:
				kind = "number"
				if !strings.ContainsAny(x.String(), ".eE") {
					key = x.String()
					if key == "-0" {
						key = "0"
					}
				}
			}
			f.Types[kind]++
			if key != "" {
				if values[p] == nil {
					values[p] = map[string]bool{}
				}
				values[p][key] = true
			}
		}
		return nil
	}
	for _, row := range rows {
		if err := walk(row, "", 0); err != nil {
			return c, err
		}
	}
	for p, f := range fields {
		f.UniqueInSample = len(rows) > 1 && len(f.Types) == 1 && len(values[p]) == len(rows)
		c.Fields = append(c.Fields, *f)
	}
	sort.Slice(c.Fields, func(i, j int) bool { return c.Fields[i].Path < c.Fields[j].Path })
	if count == len(rows) {
		c.CompositeKeys = suggestPairs(c.Fields, rows)
	}
	return c, nil
}

// Suggest only minimal pairs: each column repeats, but their combination is unique.
func suggestPairs(fields []FieldProfile, rows []map[string]any) [][]string {
	type column struct {
		path  string
		codes []int
	}
	var columns []column
	for _, field := range fields {
		if field.UniqueInSample || field.Missing != 0 || len(field.Types) != 1 ||
			(field.Types["string"] == 0 && field.Types["number"] == 0) {
			continue
		}
		path, err := tokens(field.Path)
		if err != nil {
			continue
		}
		codes := make([]int, 0, len(rows))
		distinct := map[string]int{}
		for _, row := range rows {
			id, err := recordID(row, [][]string{path})
			if err != nil {
				break
			}
			code, ok := distinct[id]
			if !ok {
				code = len(distinct)
				distinct[id] = code
			}
			codes = append(codes, code)
		}
		if len(codes) == len(rows) && len(distinct) > 1 && len(distinct) < len(rows) {
			columns = append(columns, column{field.Path, codes})
			if len(columns) == 16 {
				break
			}
		}
	}
	var pairs [][]string
	for i, col := range columns {
		for _, other := range columns[i+1:] {
			seen := map[[2]int]bool{}
			for j, code := range col.codes {
				pair := [2]int{code, other.codes[j]}
				if seen[pair] {
					break
				}
				seen[pair] = true
			}
			if len(seen) == len(rows) {
				pairs = append(pairs, []string{col.path, other.path})
				if len(pairs) == 4 {
					return pairs
				}
			}
		}
	}
	return pairs
}
