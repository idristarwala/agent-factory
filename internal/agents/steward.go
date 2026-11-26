package agents

import (
	"fmt"
	"github.com/google/uuid"

	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// StubSteward is a minimal Steward for early testing.
// It takes a goal string and produces a single unit of work.
// Later it will inspect the graph and design a real swarm.

type StubSteward struct {
	store unitofwork.Store
}

func NewStubSteward(store unitofwork.Store) *StubSteward {
	return &StubSteward{store: store}
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
	return nil, fmt.Errorf("not implemented in stub")
}

func (s *StubSteward) SetAutonomyMode(mode string) error {
	return nil
}
