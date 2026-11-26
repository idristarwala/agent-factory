package agents

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/domain"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// SmartSteward uses domain packs to turn a goal into units of work
// and can create workers when given a gateway and graph.
type SmartSteward struct {
	store   unitofwork.Store
	packs   []domain.Pack
	Gateway models.Gateway
	Graph   graph.Graph
	Rules   []Rule
}

func NewSmartSteward(store unitofwork.Store) *SmartSteward {
	return &SmartSteward{
		store: store,
		packs: domain.All(),
	}
}

func (s *SmartSteward) SetRuntime(gw models.Gateway, g graph.Graph, rules []Rule) {
	s.Gateway = gw
	s.Graph = g
	s.Rules = rules
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
		chosen = s.packs[0]
	}

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
	if s.Gateway == nil || s.Graph == nil {
		return nil, fmt.Errorf("CreateWorker requires SetRuntime(gateway, graph, rules) first")
	}
	return NewLLMWorker(spec.Name, spec.Role, s.Gateway, s.Graph, s.Rules), nil
}

func (s *SmartSteward) SetAutonomyMode(mode string) error {
	return nil
}
