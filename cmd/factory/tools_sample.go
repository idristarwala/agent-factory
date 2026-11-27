package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/idristarwala/agent-factory/internal/tools"
)

func sampleTool(ctx context.Context, config, pack string, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "" {
		return fmt.Errorf("usage: factory tools sample SOURCE [PARAM=VALUE ...]")
	}
	if config == "" || pack == "" {
		return fmt.Errorf("set FACTORY_SOURCE_CONFIG and FACTORY_PACK")
	}
	params := make(map[string]string)
	for _, arg := range args[1:] {
		name, value, ok := strings.Cut(arg, "=")
		if !ok || name == "" {
			return fmt.Errorf("expected a query parameter in PARAM=VALUE form")
		}
		if _, duplicate := params[name]; duplicate {
			return fmt.Errorf("duplicate query parameter %q", name)
		}
		params[name] = value
	}
	sampler, err := tools.LoadSourceSampler(config)
	if err != nil {
		return err
	}
	raw, err := sampler.Sample(ctx, pack, args[0], params)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(raw))
	return err
}
