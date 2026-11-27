package agents

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// StubSteward is a minimal Steward. CreateWorker returns a real LLMWorker.
type StubSteward struct {
	Registry *Registry
	store    unitofwork.Store
	gateway  models.Gateway
	graph    graph.Graph
}

func NewStubSteward(store unitofwork.Store) *StubSteward {
	return &StubSteward{Registry: NewRegistry(), store: store}
}

func (s *StubSteward) SetDependencies(gw models.Gateway, g graph.Graph) {
	s.gateway = gw
	s.graph = g
}

func (s *StubSteward) InterpretGoal(goal string, graphSnapshot any) ([]*unitofwork.UnitOfWork, error) {
	id := uuid.NewString()
	uow := &unitofwork.UnitOfWork{
		ID:             id,
		GoalID:         "goal-" + id[:8],
		Description:    fmt.Sprintf("Process goal: %s", goal),
		ExpectedOutput: "graph mutation addressing the goal",
		Mode:           "non_deterministic",
		Budget:         map[string]any{"max_tokens": 2000},
	}
	if err := s.store.Create(uow); err != nil {
		return nil, err
	}
	return []*unitofwork.UnitOfWork{uow}, nil
}

func (s *StubSteward) CreateWorker(spec WorkerSpec) (Worker, error) {
	if s.gateway == nil || s.graph == nil {
		return nil, fmt.Errorf("steward dependencies not set; call SetDependencies")
	}
	def, err := s.Registry.Resolve(AgentVersion{Name: spec.Name, Role: spec.Role, ModelTier: spec.ModelTier, Tools: spec.Tools, CreatedBy: "steward"})
	if err != nil {
		return nil, err
	}
	if def.ModelTier != "" && def.ModelTier != "cheap" {
		return nil, fmt.Errorf("unsupported worker model tier %q", def.ModelTier)
	}
	worker := NewLLMWorker(def.Name, def.Role, s.gateway, s.graph, nil)
	worker.Definition = def
	return worker, nil
}

func (s *StubSteward) SetAutonomyMode(mode string) error {
	return nil
}
