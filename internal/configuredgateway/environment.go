package configuredgateway

import (
	"cmp"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/idristarwala/agent-factory/internal/obs"
)

type environmentProvider struct {
	protocol, base, key, cheap, frontier, schemaMode string
	timeout                                          time.Duration
}

func environmentRouter() (*Router, error) {
	name := strings.ToLower(os.Getenv("LLM_PROVIDER"))
	if name == "" {
		name = "openai"
		if os.Getenv("LLM_BASE_URL") == "" && (os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("LLM_API_KEY") != "") {
			name = "gemini"
		}
	}
	p, err := providerFromEnv(name)
	if err != nil {
		return nil, err
	}
	router := &Router{routes: map[string]route{}}
	retry, err := environmentRetry()
	if err != nil {
		return nil, err
	}
	logger := obs.New()
	add := func(name, model string, p environmentProvider) error {
		client := &http.Client{Timeout: p.timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("gateway endpoint redirected") }}
		m := modelConfig{Retry: retry, Protocol: p.protocol, Model: model, SchemaMode: p.schemaMode}
		entry, err := buildRoute(m, p.base, p.key, nil, client, logger)
		if err == nil {
			router.routes[name] = entry
		}
		return err
	}
	if err := add("cheap", p.cheap, p); err != nil {
		return nil, err
	}
	if err := add("frontier", p.frontier, p); err != nil {
		return nil, err
	}
	router.routes["council"] = router.routes["frontier"]
	judge, model := strings.ToLower(os.Getenv("LLM_JUDGE_PROVIDER")), os.Getenv("LLM_JUDGE_MODEL")
	if judge == "" && model == "" {
		return router, nil
	}
	if judge == "" || model == "" {
		return nil, fmt.Errorf("set both LLM_JUDGE_PROVIDER and LLM_JUDGE_MODEL")
	}
	var j environmentProvider
	switch judge {
	case "gemini":
		j = environmentProvider{protocol: "gemini", base: "https://generativelanguage.googleapis.com/v1beta", key: os.Getenv("GEMINI_API_KEY")}
	case "openrouter":
		j = environmentProvider{protocol: "openai", base: "https://openrouter.ai/api/v1", key: os.Getenv("OPENROUTER_API_KEY"), schemaMode: "non_strict"}
	default:
		return nil, fmt.Errorf("unknown judge provider %q", judge)
	}
	if strings.TrimSpace(j.key) == "" {
		return nil, fmt.Errorf("missing dedicated credential for %s judge", judge)
	}
	j.timeout = time.Minute
	if err := add("judge", model, j); err != nil {
		return nil, err
	}
	return router, nil
}

func providerFromEnv(name string) (environmentProvider, error) {
	var p environmentProvider
	switch name {
	case "gemini":
		p.protocol = "gemini"
		p.base = "https://generativelanguage.googleapis.com/v1beta"
		p.key = cmp.Or(os.Getenv("GEMINI_API_KEY"), os.Getenv("LLM_API_KEY"))
		p.cheap = cmp.Or(os.Getenv("GEMINI_CHEAP_MODEL"), "gemini-3.5-flash-lite")
		p.frontier = cmp.Or(os.Getenv("GEMINI_FRONTIER_MODEL"), "gemini-3.5-flash-lite")
		p.timeout = 90 * time.Second
	case "anthropic":
		p.protocol = "anthropic"
		p.base = "https://api.anthropic.com"
		p.key = os.Getenv("ANTHROPIC_API_KEY")
		p.schemaMode = "forced_tool"
		p.cheap = cmp.Or(os.Getenv("ANTHROPIC_CHEAP_MODEL"), "claude-3-haiku-20240307")
		p.frontier = cmp.Or(os.Getenv("ANTHROPIC_FRONTIER_MODEL"), "claude-3-5-sonnet-20241022")
		p.timeout = 90 * time.Second
	case "openrouter", "openai", "openai-compat", "groq":
		p.protocol = "openai"
		p.base = os.Getenv("LLM_BASE_URL")
		p.key = os.Getenv("LLM_API_KEY")
		p.schemaMode = "non_strict"
		p.timeout = time.Minute
		defaultCheap := "llama-3.1-8b-instant"
		if name == "openrouter" && p.base == "" {
			p.base = "https://openrouter.ai/api/v1"
			p.key = os.Getenv("OPENROUTER_API_KEY")
			defaultCheap = "gpt-4o-mini"
		}
		p.cheap = cmp.Or(os.Getenv("LLM_CHEAP_MODEL"), defaultCheap)
		p.frontier = cmp.Or(os.Getenv("LLM_FRONTIER_MODEL"), p.cheap)
	default:
		return p, fmt.Errorf("unknown LLM_PROVIDER %q", name)
	}
	if strings.TrimSpace(p.key) == "" || p.base == "" {
		return p, fmt.Errorf("missing endpoint or credentials for %s", name)
	}
	if !validGatewayURL(p.base) {
		return p, fmt.Errorf("invalid gateway endpoint for %s", name)
	}
	return p, nil
}
