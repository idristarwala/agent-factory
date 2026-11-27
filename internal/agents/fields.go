package agents

import (
	"fmt"
	"math"
	"slices"
	"strings"

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

// continuousFields resolves a write scope before the steward queues the unit.
func continuousFields(goal string, configured []string, current map[string]any) ([]string, error) {
	fields := configured
	// This compact task form names one literal field. More complex goals can
	// carry their precise scope separately from the natural-language prompt.
	if verb, rest, ok := strings.Cut(strings.TrimSpace(goal), " "); ok {
		name := strings.TrimSpace(rest)
		if only, remainder, found := strings.Cut(name, " "); found && strings.EqualFold(only, "only") {
			name = strings.TrimSpace(remainder)
		}
		if slices.Contains([]string{"edit", "fix", "update", "correct"}, strings.ToLower(verb)) && name != "" && !strings.ContainsAny(name, " \t\n") {
			if len(configured) > 0 && !slices.Contains(configured, name) {
				return nil, fmt.Errorf("task field %q is outside the configured write scope", name)
			}
			fields = []string{name}
		}
	}
	if len(fields) == 0 {
		for name, value := range current {
			if name == "status" || fieldType(value) == "" {
				continue
			}
			if len(fields) > 0 {
				return nil, fmt.Errorf("ambiguous write scope: supply editable fields for a multi-field record")
			}
			fields = []string{name}
		}
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("record has no editable fields")
	}
	for _, name := range fields {
		if name == "status" || fieldType(current[name]) == "" {
			return nil, fmt.Errorf("invalid editable field %q", name)
		}
	}
	return slices.Clone(fields), nil
}

func outputFields(u *unitofwork.UnitOfWork, current map[string]any) ([]string, map[string]string, error) {
	allowed := u.AllowedFields
	if len(allowed) == 0 {
		fields, err := continuousFields(u.Description, nil, current)
		if err != nil {
			return nil, nil, err
		}
		allowed = fields
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
