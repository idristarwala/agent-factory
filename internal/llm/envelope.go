package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OpenAIEnvelope nests the standard request under ParamsKey alongside the
// configured template. A string value of "{model}" expands to the wire model.
type OpenAIEnvelope struct {
	Template  map[string]any `yaml:"template" json:"template"`
	ParamsKey string         `yaml:"params_key" json:"params_key"`
}

func (e *OpenAIEnvelope) Validate() error {
	if e == nil || strings.TrimSpace(e.ParamsKey) == "" {
		return fmt.Errorf("openai-envelope needs params_key")
	}
	if _, exists := e.Template[e.ParamsKey]; exists {
		return fmt.Errorf("params_key conflicts with template field %q", e.ParamsKey)
	}
	return nil
}

func (a *OpenAIAdapter) requestBody(req ChatRequest, stream bool) ([]byte, error) {
	params, err := buildOpenAIBody(req, stream)
	if err != nil || a.Envelope == nil {
		return params, err
	}
	if err := a.Envelope.Validate(); err != nil {
		return nil, &Error{Type: "BAD_REQUEST", Message: err.Error()}
	}
	out := make(map[string]any, len(a.Envelope.Template)+1)
	for key, value := range a.Envelope.Template {
		out[key] = expandModel(value, req.Model)
	}
	out[a.Envelope.ParamsKey] = json.RawMessage(params)
	return json.Marshal(out)
}

func expandModel(value any, model string) any {
	switch v := value.(type) {
	case string:
		if v == "{model}" {
			return model
		}
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = expandModel(item, model)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = expandModel(item, model)
		}
		return out
	}
	return value
}
