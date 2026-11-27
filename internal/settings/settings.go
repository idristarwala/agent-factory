package settings

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
)

//go:embed workspace.schema.json
var Schema json.RawMessage

type ContinuousLimits struct {
	WorkerCount      int `json:"worker_count,omitempty"`
	MaxFailures      int `json:"max_failures,omitempty"`
	StagnantRounds   int `json:"stagnant_rounds,omitempty"`
	JudgeEvaluations int `json:"judge_evaluations,omitempty"`
	JudgeApprovals   int `json:"judge_approvals,omitempty"`
}

type Settings struct {
	ContinuousLimits
	MaxRounds  int  `json:"max_rounds"`
	MaxTokens  int  `json:"max_tokens"`
	MaxMinutes int  `json:"max_minutes"`
	ReplayGate bool `json:"replay_gate"`
	Observe    bool `json:"observe"`
}

const ID = "settings/workspace"

func Current(s revision.Store) (Settings, int, error) {
	h, err := revision.Documents(s, ID)
	if err != nil {
		return Settings{}, 0, err
	}
	v := Settings{MaxRounds: 10, MaxTokens: 1000000, MaxMinutes: 30, ContinuousLimits: ContinuousLimits{
		WorkerCount: 2, MaxFailures: 20, StagnantRounds: 2, JudgeEvaluations: 2, JudgeApprovals: 2,
	}}
	if len(h) == 0 {
		return v, 0, nil
	}
	d := h[len(h)-1]
	err = json.Unmarshal(d.Data, &v)
	return v, d.Version, err
}
func (v Settings) Validate() error {
	if v.MaxRounds <= 0 || v.MaxTokens <= 0 || v.MaxMinutes <= 0 || v.MaxMinutes > 525600 {
		return fmt.Errorf("limits must be positive and wall time must be at most one year")
	}
	if v.WorkerCount < 1 || v.WorkerCount > 256 {
		return fmt.Errorf("worker_count must be between 1 and 256")
	}
	if v.MaxFailures < 1 || v.StagnantRounds < 1 {
		return fmt.Errorf("max_failures and stagnant_rounds must be positive")
	}
	if v.JudgeEvaluations < 1 || v.JudgeEvaluations > 32 || v.JudgeApprovals < 1 || v.JudgeApprovals > v.JudgeEvaluations {
		return fmt.Errorf("judge_evaluations must be between 1 and 32 and judge_approvals between 1 and judge_evaluations")
	}
	return nil
}
