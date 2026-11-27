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

// InterpretGoal returns units only for entities that still need work.
// Returns empty when the graph is clean so the runner can stop.
func (s *SmartSteward) InterpretGoal(goal string, graphSnapshot any) ([]*unitofwork.UnitOfWork, error) {
	// Check graph for remaining problems
	if s.Graph != nil {
		snap, ok := graphSnapshot.(map[graph.NodeID]*graph.Node)
		if !ok {
			snap = s.Graph.Snapshot()
		}
		var needsWork []*graph.Node
		for _, n := range snap {
			if n == nil {
				continue
			}
			status, _ := n.Props["status"].(string)
			if status != "fixed" && status != "clean" && status != "complete" {
				needsWork = append(needsWork, n)
			}
		}
		if len(needsWork) == 0 {
			return nil, nil // graph is clean
		}
		// Create units only for nodes that need work
		var uows []*unitofwork.UnitOfWork
		for _, n := range needsWork {
			id := uuid.NewString()
			u := &unitofwork.UnitOfWork{
				ID:             id,
				GoalID:         "goal-" + id[:8],
				Description:    fmt.Sprintf("%s — fix %s", goal, n.ID),
				InputSubgraph:  n.ID,
				ExpectedOutput: "graph mutation",
				Mode:           "non_deterministic",
			}
			if err := s.store.Create(u); err != nil {
				return nil, err
			}
			uows = append(uows, u)
		}
		return uows, nil
	}

	// Fallback when no graph is set
	goalLower := strings.ToLower(goal)
	var chosen domain.Pack
	for _, p := range s.packs {
		if strings.Contains(goalLower, p.Name) || strings.Contains(goalLower, "title") || strings.Contains(goalLower, "description") || strings.Contains(goalLower, "product") {
			chosen = p
			break
		}
	}
	if chosen.Name == "" && len(s.packs) > 0 {
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
		}
		if err := s.store.Create(u); err != nil {
			return nil, err
		}
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
