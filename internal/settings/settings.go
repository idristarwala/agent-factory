package settings

import (
	"encoding/json"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
)

type Settings struct {
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
	if len(h) == 0 {
		return Settings{MaxRounds: 10, MaxTokens: 1000000, MaxMinutes: 30}, 0, nil
	}
	d := h[len(h)-1]
	var v Settings
	err = json.Unmarshal(d.Data, &v)
	return v, d.Version, err
}
func (v Settings) Validate() error {
	if v.MaxRounds <= 0 || v.MaxTokens <= 0 || v.MaxMinutes <= 0 || v.MaxMinutes > 525600 {
		return fmt.Errorf("limits must be positive and wall time must be at most one year")
	}
	return nil
}
