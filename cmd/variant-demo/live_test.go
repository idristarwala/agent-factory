package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/idristarwala/agent-factory/internal/llm"
)

type membershipClient struct {
	llm.Client
	prompt string
	reply  string
}

func (m *membershipClient) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	m.prompt = req.Messages[0].Content
	if req.Model != "cheap" || req.ResponseSchema == nil {
		return nil, fmt.Errorf("missing route or schema")
	}
	return &llm.ChatResponse{Content: m.reply}, nil
}

func TestLiveBenchmarkRequiresCompleteDecisions(t *testing.T) {
	data := []byte(inputs + `,"labels":[{"id":"1","group":"hidden-label"},{"id":"2","group":"hidden-label"}]}`)
	decision := `{"entity_id":"1","group_id":"x","same_product":true,"reason":"source"}`
	client := &membershipClient{reply: `{"decisions":[` + decision + `,` + strings.Replace(decision, `"1"`, `"2"`, 1) + `]}`}
	var out bytes.Buffer
	if err := benchmark(data, &out, client); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "configured model") || strings.Contains(client.prompt, "hidden-label") || !strings.Contains(client.prompt, `"group":"x"`) {
		t.Fatal(out.String(), client.prompt)
	}
	for _, response := range []string{"broken", `{"decisions":[]}`, `{"decisions":[` + decision + `,` + decision + `]}`, `{"decisions":[{"entity_id":"1"}]}`} {
		client.reply = response
		out.Reset()
		if err := benchmark(data, &out, client); err == nil || out.Len() != 0 {
			t.Fatal(response, out.String(), err)
		}
	}
}
