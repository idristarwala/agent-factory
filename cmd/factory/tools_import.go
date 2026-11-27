package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

type importedTool struct {
	Method     string            `yaml:"method"`
	URL        string            `yaml:"url"`
	SideEffect string            `yaml:"side_effect"`
	Query      []string          `yaml:"query,omitempty"`
	Required   []string          `yaml:"required,omitempty"`
	Auth       map[string]string `yaml:"auth,omitempty"`
}

func importTools(args []string, out, diagnostics io.Writer) error {
	if len(args) == 0 || args[0] != "import" {
		return fmt.Errorf("usage: factory tools import [flags] FILE")
	}
	flags := flag.NewFlagSet("tools import", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	selected := flags.String("operation", "", "import a single operationId")
	base := flags.String("server", "", "fixed HTTP(S) server base URL")
	key := flags.String("key-file", "", "credential file reference for one selected operation")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 1 || (*key != "" && *selected == "") {
		return fmt.Errorf("supply one OpenAPI file; -key-file requires -operation")
	}
	data, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		return err
	}
	doc, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return fmt.Errorf("read OpenAPI: %w", err)
	}
	if !strings.HasPrefix(doc.OpenAPI, "3.0.") && !strings.HasPrefix(doc.OpenAPI, "3.1.") {
		return fmt.Errorf("expected OpenAPI 3.0 or 3.1")
	}
	if err := doc.Validate(context.Background()); err != nil {
		return fmt.Errorf("invalid OpenAPI document: %w", err)
	}
	defs := map[string]importedTool{}
	for _, path := range slices.Sorted(maps.Keys(doc.Paths.Map())) {
		item := doc.Paths.Value(path)
		operations := item.Operations()
		for _, method := range slices.Sorted(maps.Keys(operations)) {
			op := operations[method]
			if *selected != "" && op.OperationID != *selected {
				continue
			}
			if method != "GET" {
				if *selected != "" {
					return fmt.Errorf("%s %s: only GET evidence operations can be imported", method, path)
				}
				fmt.Fprintf(diagnostics, "Skipped %s %s (non-GET operation)\n", method, path)
				continue
			}
			name := op.OperationID
			if name == "" {
				name = "GET " + path
			}
			if _, duplicate := defs[name]; duplicate {
				return fmt.Errorf("duplicate tool name %q", name)
			}
			def, err := convertOperation(doc, item, op, path, *base, *key)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			defs[name] = def
		}
	}
	if len(defs) == 0 {
		return fmt.Errorf("no matching GET operation")
	}
	result, err := yaml.Marshal(map[string]any{"tools": defs})
	if err != nil {
		return err
	}
	_, err = out.Write(result)
	return err
}

func convertOperation(doc *openapi3.T, item *openapi3.PathItem, op *openapi3.Operation, path, base, key string) (importedTool, error) {
	var tool importedTool
	if base == "" {
		servers := doc.Servers
		if len(item.Servers) > 0 {
			servers = item.Servers
		}
		if op.Servers != nil {
			servers = *op.Servers
		}
		if len(servers) != 1 {
			return tool, fmt.Errorf("select an absolute server with -server")
		}
		base = servers[0].URL
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || strings.ContainsAny(base, "{}?#\\") {
		return tool, fmt.Errorf("supply a fixed absolute HTTP(S) server without credentials, query or fragment")
	}
	if op.RequestBody != nil {
		return tool, fmt.Errorf("GET request bodies require manual configuration")
	}
	tool.Method = "GET"
	tool.SideEffect = "read-only"
	tool.URL = strings.TrimRight(base, "/") + path
	// OpenAPI operation-level parameters override matching path-level parameters.
	params := map[string]*openapi3.Parameter{}
	for _, list := range []openapi3.Parameters{item.Parameters, op.Parameters} {
		for _, ref := range list {
			p := ref.Value
			params[p.In+" "+p.Name] = p
		}
	}
	for _, k := range slices.Sorted(maps.Keys(params)) {
		p := params[k]
		if p.Schema == nil || p.Schema.Value == nil || p.Schema.Value.Type == nil || len(*p.Schema.Value.Type) != 1 || !slices.Contains([]string{"string", "integer", "number", "boolean"}, (*p.Schema.Value.Type)[0]) {
			return tool, fmt.Errorf("parameter %q needs a scalar schema", p.Name)
		}
		switch p.In {
		case "query":
			if (p.Style != "" && p.Style != "form") || p.AllowReserved {
				return tool, fmt.Errorf("unsupported serialization for %q", p.Name)
			}
			tool.Query = append(tool.Query, p.Name)
		case "path":
			if p.Style != "" && p.Style != "simple" {
				return tool, fmt.Errorf("unsupported path style for %q", p.Name)
			}
		default:
			return tool, fmt.Errorf("parameter %q in %s requires manual configuration", p.Name, p.In)
		}
		if p.Required {
			tool.Required = append(tool.Required, p.Name)
		}
	}
	security := doc.Security
	if op.Security != nil {
		security = *op.Security
	}
	if len(security) == 0 || len(security) == 1 && len(security[0]) == 0 {
		if key != "" {
			return tool, fmt.Errorf("credential file supplied for an operation without authentication")
		}
		return tool, nil
	}
	if len(security) != 1 || len(security[0]) != 1 || key == "" {
		return tool, fmt.Errorf("select one operation using bearer or header API-key authentication and supply -key-file")
	}
	for name := range security[0] {
		if doc.Components == nil {
			return tool, fmt.Errorf("unknown security scheme %q", name)
		}
		ref := doc.Components.SecuritySchemes[name]
		if ref == nil || ref.Value == nil {
			return tool, fmt.Errorf("unknown security scheme %q", name)
		}
		scheme := ref.Value
		switch {
		case scheme.Type == "http" && strings.EqualFold(scheme.Scheme, "bearer"):
			tool.Auth = map[string]string{"type": "bearer", "key_file": key}
		case scheme.Type == "apiKey" && scheme.In == "header":
			tool.Auth = map[string]string{"type": "header", "name": scheme.Name, "key_file": key}
		default:
			return tool, fmt.Errorf("unsupported security scheme %q", name)
		}
	}
	return tool, nil
}
