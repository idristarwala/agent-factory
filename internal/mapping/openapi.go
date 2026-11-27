package mapping

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// APIResponse selects the documented JSON response independently of a sample.
type APIResponse struct {
	Document any    `json:"document"`
	Path     string `json:"path"`
	Method   string `json:"method"`
	Status   string `json:"status"`
}
type DeclaredField struct {
	Path     string   `json:"path"`
	Types    []string `json:"types,omitempty"`
	Required bool     `json:"required"`
}

func object(v any) map[string]any { m, _ := v.(map[string]any); return m }
func child(v any, keys ...string) any {
	for _, k := range keys {
		v = object(v)[k]
	}
	return v
}

func (a APIResponse) Fields(collection string) ([]DeclaredField, error) {
	doc := a.Document
	if raw, ok := doc.(string); ok {
		if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
			return nil, fmt.Errorf("invalid OpenAPI document")
		}
	}
	version, _ := child(doc, "openapi").(string)
	if !strings.HasPrefix(version, "3.0.") && !strings.HasPrefix(version, "3.1.") {
		return nil, fmt.Errorf("supply an OpenAPI 3.0 or 3.1 document")
	}
	resolve := func(v any) (map[string]any, error) {
		for depth := 0; depth < 32; depth++ {
			m := object(v)
			if m == nil {
				return nil, fmt.Errorf("missing OpenAPI response or schema")
			}
			ref, has := m["$ref"]
			if !has {
				return m, nil
			}
			p, ok := ref.(string)
			if !ok || !strings.HasPrefix(p, "#/") {
				return nil, fmt.Errorf("OpenAPI references must be local JSON pointers")
			}
			target, found, err := SourceValue(doc, p[1:])
			if err != nil || !found {
				return nil, fmt.Errorf("unresolved OpenAPI reference")
			}
			v = target
		}
		return nil, fmt.Errorf("OpenAPI reference depth exceeded")
	}
	method := strings.ToLower(a.Method)
	if method == "" {
		method = "get"
	}
	status := a.Status
	if status == "" {
		status = "200"
	}
	resp, err := resolve(child(doc, "paths", a.Path, method, "responses", status))
	if err != nil {
		return nil, err
	}
	schema, err := resolve(child(resp, "content", "application/json", "schema"))
	if err != nil {
		return nil, err
	}
	parts, err := tokens(collection)
	if err != nil {
		return nil, err
	}
	for _, p := range parts {
		schema, err = resolve(child(schema, "properties", p))
		if err != nil {
			return nil, fmt.Errorf("collection not found in declared response: %w", err)
		}
	}
	if items, ok := schema["items"]; ok {
		schema, err = resolve(items)
		if err != nil {
			return nil, err
		}
	}
	var fields []DeclaredField
	var walk func(map[string]any, string, int) error
	walk = func(s map[string]any, parent string, depth int) error {
		if depth > 32 {
			return fmt.Errorf("declared schema exceeds nesting limit")
		}
		for _, keyword := range []string{"oneOf", "anyOf", "allOf"} {
			if _, ok := s[keyword]; ok {
				return fmt.Errorf("unsupported composed OpenAPI schema at %q", parent)
			}
		}
		required := map[string]bool{}
		if keys, ok := s["required"].([]any); ok {
			for _, k := range keys {
				if name, ok := k.(string); ok {
					required[name] = true
				}
			}
		}
		props := object(s["properties"])
		for k, v := range props {
			path := pointer(parent, k)
			if len(fields) >= 512 || len(path) > 1024 {
				return fmt.Errorf("declared schema exceeds field limits")
			}
			f, err := resolve(v)
			if err != nil {
				return err
			}
			var kinds []string
			switch typ := f["type"].(type) {
			case string:
				kinds = []string{typ}
			case []any:
				for _, t := range typ {
					if name, ok := t.(string); ok {
						kinds = append(kinds, name)
					}
				}
			default:
				if f["properties"] != nil {
					kinds = []string{"object"}
				} else if f["items"] != nil {
					kinds = []string{"array"}
				}
			}
			if nullable, _ := f["nullable"].(bool); nullable && !slices.Contains(kinds, "null") {
				kinds = append(kinds, "null")
			}
			for _, kind := range kinds {
				if !slices.Contains([]string{"string", "integer", "number", "array", "object", "boolean", "null"}, kind) {
					return fmt.Errorf("invalid schema type at %q", path)
				}
			}
			sort.Strings(kinds)
			fields = append(fields, DeclaredField{path, kinds, required[k]})
			if err := walk(f, path, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(schema, "", 0); err != nil {
		return nil, err
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Path < fields[j].Path })
	return fields, nil
}
