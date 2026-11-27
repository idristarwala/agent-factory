package main

import "github.com/idristarwala/agent-factory/internal/factory"

type usageSummary struct {
	Tokens       int            `json:"tokens_used"`
	RunTokens    int            `json:"run_tokens"`
	ChatTokens   int            `json:"non_run_tokens"`
	Input        int            `json:"input_tokens"`
	Output       int            `json:"output_tokens"`
	ModelCalls   int            `json:"model_calls"`
	Failed       int            `json:"failed_model_calls"`
	ToolCalls    int            `json:"tool_calls"`
	Tools        map[string]int `json:"tool_calls_by_name"`
	EstimatedUSD float64        `json:"estimated_usd"`
	Unpriced     int            `json:"unpriced_calls"`
}

func runUsage(runs []factory.Run, id string) usageSummary {
	s := usageSummary{Tools: map[string]int{}}
	for _, r := range runs {
		if id != "" && r.ID != id {
			continue
		}
		for tool, n := range r.Tools {
			s.Tools[tool] += n
			s.ToolCalls += n
		}
		for _, c := range r.Calls {
			s.ModelCalls++
			if c.Failed {
				s.Failed++
				continue
			}
			s.RunTokens += c.Total
			s.Input += c.Input
			s.Output += c.Output
			s.EstimatedUSD += c.Cost
			if !c.Priced {
				s.Unpriced++
			}
		}
	}
	s.Tokens = s.RunTokens
	return s
}
