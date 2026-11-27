package agents

import (
	"fmt"
	"math"
	"sort"

	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

func fieldType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int32, int64, uint, uint32, uint64:
		return "integer"
	case float64, float32:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return ""
}
func outputFields(u *unitofwork.UnitOfWork, current map[string]any) ([]string, map[string]string, error) {
	allowed := u.AllowedFields
	if len(allowed) == 0 {
		for k, v := range current {
			if k != "status" && fieldType(v) != "" {
				allowed = append(allowed, k)
			}
		}
		sort.Strings(allowed)
	}
	if len(allowed) == 0 {
		return nil, nil, fmt.Errorf("record has no editable fields")
	}
	types := make(map[string]string, len(allowed))
	for _, k := range allowed {
		typ := u.FieldTypes[k]
		if typ == "" {
			typ = fieldType(current[k])
		}
		if typ == "" {
			typ = "string"
		}
		switch typ {
		case "string", "boolean", "integer", "number", "array", "object":
		default:
			return nil, nil, fmt.Errorf("unsupported field type for %s", k)
		}
		types[k] = typ
	}
	for _, k := range u.RequiredFields {
		if _, ok := types[k]; !ok {
			return nil, nil, fmt.Errorf("required field %s is not editable", k)
		}
	}
	return allowed, types, nil
}
func validateFields(values map[string]any, types map[string]string, required []string) error {
	for _, k := range required {
		if _, ok := values[k]; !ok {
			return fmt.Errorf("missing required field %s", k)
		}
	}
	for k, v := range values {
		actual := fieldType(v)
		if types[k] == actual {
			continue
		}
		if n, ok := v.(float64); ok && types[k] == "integer" && math.Trunc(n) == n {
			continue
		}
		return fmt.Errorf("expected %s for %s", types[k], k)
	}
	return nil
}
