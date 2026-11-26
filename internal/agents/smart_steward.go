package agents

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/domain"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// SmartSteward uses domain packs to turn a goal into units of work
// and can create workers.
type SmartSteward struct {
	store unitofwork.Store
	packs []domain.Pack
}

func NewSmartSteward(store unitofwork.Store) *SmartSteward {
	return &SmartSteward{
		store: store,
		packs: domain.All(),
	}
}

func (s *SmartSteward) InterpretGoal(goal string, graphSnapshot any) ([]*unitofwork.UnitOfWork, error) {
	goalLower := strings.ToLower(goal)
	var chosen domain.Pack
	for _, p := range s.packs {
		if strings.Contains(goalLower, p.Name) || strings.Contains(goalLower, "title") || strings.Contains(goalLower, "description") || strings.Contains(goalLower, "product") {
			chosen = p
			break
		}
	}
	if chosen.Name == "" {
		chosen = s.packs[0] // default to catalog
	}

	// Create one unit of work per worker role (simple strategy)
	var uows []*unitofwork.UnitOfWork
	for _, role := range chosen.WorkerRoles {
		id := uuid.NewString()
		u := &unitofwork.UnitOfWork{
			ID:             id,
			GoalID:         "goal-" + id[:8],
			Description:    fmt.Sprintf("[%s] %s — role: %s", chosen.Name, goal, role),
			ExpectedOutput: "graph mutation",
			Mode:           "non_deterministic",
			Budget:         map[string]any{"max_tokens": 1000},
		}
		if err := s.store.Create(u); err != nil {
			return nil, err
		}
		uows = append(uows, u)
	}
	if len(uows) == 0 {
		// fallback single unit
		id := uuid.NewString()
		u := &unitofwork.UnitOfWork{
			ID:          id,
			Description: goal,
			Mode:        "non_deterministic",
		}
		s.store.Create(u)
		uows = append(uows, u)
	}
	return uows, nil
}

func (s *SmartSteward) CreateWorker(spec WorkerSpec) (Worker, error) {
	// Actual worker creation with gateway happens outside; this returns a marker.
	return nil, fmt.Errorf("use NewLLMWorker directly; Steward records the intent")
}

func (s *SmartSteward) SetAutonomyMode(mode string) error {
	return nil
}
