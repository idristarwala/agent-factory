package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Compile the caller's schema before contacting a gateway. Local references
// resolve within the supplied document without reading files or fetching URLs.
func compileOutputSchema(raw map[string]any) (*jsonschema.Schema, error) {
	if raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, &Error{Type: "BAD_REQUEST", Message: "response schema must be JSON"}
	}
	compiler := jsonschema.NewCompiler()
	compiler.LoadURL = func(string) (io.ReadCloser, error) {
		return nil, errors.New("external schema references are unsupported")
	}
	const resource = "https://schema.invalid/response.json"
	if err = compiler.AddResource(resource, bytes.NewReader(data)); err != nil {
		return nil, &Error{Type: "BAD_REQUEST", Message: "invalid response schema", Cause: err}
	}
	schema, err := compiler.Compile(resource)
	if err != nil {
		return nil, &Error{Type: "BAD_REQUEST", Message: "invalid response schema", Cause: err}
	}
	return schema, nil
}

func validateOutput(schema *jsonschema.Schema, content string) error {
	if schema == nil {
		return nil
	}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return &Error{Type: "BAD_RESPONSE", Message: "structured response must contain JSON"}
	}
	if err := decoder.Decode(&value); err != io.EOF {
		return &Error{Type: "BAD_RESPONSE", Message: "structured response must contain one JSON value"}
	}
	if err := schema.Validate(value); err != nil {
		return &Error{Type: "BAD_RESPONSE", Message: "structured response violates the requested schema"}
	}
	return nil
}
