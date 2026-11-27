package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/theory/jsonpath"
)

type responseConfig struct {
	Root string
	Map  map[string]string
}

type responseMapping struct {
	root   *jsonpath.Path
	fields map[string]*jsonpath.Path
}

func (c responseConfig) compile() (*responseMapping, error) {
	root := c.Root
	if root == "" {
		root = "$"
	}
	path, err := jsonpath.Parse(root)
	if err != nil {
		return nil, fmt.Errorf("invalid response root: %w", err)
	}
	m := &responseMapping{root: path}
	if c.Map != nil {
		m.fields = make(map[string]*jsonpath.Path, len(c.Map))
		for name, expr := range c.Map {
			if strings.TrimSpace(name) == "" {
				return nil, errors.New("response field name is empty")
			}
			path, err := jsonpath.Parse(expr)
			if err != nil {
				return nil, fmt.Errorf("invalid response field %q: %w", name, err)
			}
			m.fields[name] = path
		}
	}
	return m, nil
}

func selectValue(path *jsonpath.Path, input any) (any, error) {
	values := path.Select(input)
	if path.Query().Singular() == nil {
		if values == nil {
			return []any{}, nil
		}
		return []any(values), nil
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("path %s is absent", path)
	}
	return values[0], nil
}

func (m *responseMapping) apply(body []byte) (string, error) {
	if !json.Valid(body) {
		return "", errors.New("mapped response requires valid JSON")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var data any
	if err := dec.Decode(&data); err != nil {
		return "", err
	}
	root, err := selectValue(m.root, data)
	if err != nil {
		return "", fmt.Errorf("response root: %w", err)
	}
	result := root
	if m.fields != nil {
		out := make(map[string]any, len(m.fields))
		for name, path := range m.fields {
			value, err := selectValue(path, root)
			if err != nil {
				return "", fmt.Errorf("response field %q: %w", name, err)
			}
			out[name] = value
		}
		result = out
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	if len(b) > maxHTTPResponse {
		return "", errors.New("mapped response exceeds 512 KiB")
	}
	return string(b), nil
}
