package tools

import (
	"encoding/json"
	"os"
)

// ToolDef is the JSON form of a tool definition.
type ToolDef struct {
	Name        string            `json:"name"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	AuthType    string            `json:"auth_type"`
	KeyFile     string            `json:"key_file"`
	Headers     map[string]string `json:"headers"`
	SideEffect  string            `json:"side_effect"`
	Retries     int               `json:"retries"`
}

// LoadTools reads a JSON file of tool definitions and returns HTTPTools.
func LoadTools(path string) (map[string]*HTTPTool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var defs []ToolDef
	if err := json.Unmarshal(data, &defs); err != nil {
		return nil, err
	}
	out := make(map[string]*HTTPTool)
	for _, d := range defs {
		t := NewHTTPTool(d.Name, d.Method, d.URL)
		t.AuthType = d.AuthType
		t.KeyFile = d.KeyFile
		t.Headers = d.Headers
		if d.SideEffect != "" {
			t.SideEffect = d.SideEffect
		}
		if d.Retries > 0 {
			t.Retries = d.Retries
		}
		out[d.Name] = t
	}
	return out, nil
}
