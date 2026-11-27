package agents

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/pack"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

// SmartSteward uses domain packs to turn a goal into units of work
// and can create workers when given a gateway and graph.
type SmartSteward struct {
	AllowedFields []string
	Registry      *Registry
	store         unitofwork.Store
	PlanningPack  *pack.Pack
	Gateway       models.Gateway
	Graph         graph.Graph
	Rules         []Rule
}

func NewSmartSteward(store unitofwork.Store) *SmartSteward {
	return &SmartSteward{Registry: NewRegistry(),
		store: store,
	}
}

func (s *SmartSteward) SetRuntime(gw models.Gateway, g graph.Graph, rules []Rule) {
	s.Gateway = gw
	s.Graph = g
	s.Rules = rules
}

func pendingRecords(snapshot map[graph.NodeID]*graph.Node) []*graph.Node {
	var needsWork []*graph.Node
	for _, n := range snapshot {
		if n == nil {
			continue
		}
		status, _ := n.Props["status"].(string)
		if status != "fixed" && status != "clean" && status != "complete" {
			needsWork = append(needsWork, n)
		}
	}

	return needsWork
}

// GoalComplete checks the same terminal states used for planning without creating work units.
func (s *SmartSteward) GoalComplete(snapshot any) bool {
	nodes, ok := snapshot.(map[graph.NodeID]*graph.Node)
	return ok && len(pendingRecords(nodes)) == 0
}

// InterpretGoal returns units only for entities that still need work.
// Returns empty when the graph is clean so the runner can stop.
func (s *SmartSteward) InterpretGoal(goal string, graphSnapshot any) ([]*unitofwork.UnitOfWork, error) {
	// Check graph for remaining problems
	if s.Graph != nil {
		var snap map[graph.NodeID]*graph.Node
		if graphSnapshot != nil {
			if s, ok := graphSnapshot.(map[graph.NodeID]*graph.Node); ok {
				snap = s
			}
		}
		if snap == nil {
			// Snapshot returns any; assert it
			if s, ok := s.Graph.Snapshot().(map[graph.NodeID]*graph.Node); ok {
				snap = s
			}
		}
		if snap == nil {
			return nil, nil
		}
		needsWork := pendingRecords(snap)
		if len(needsWork) == 0 {
			return nil, nil // graph is clean
		}
		// Create units only for nodes that need work
		var uows []*unitofwork.UnitOfWork
		for _, n := range needsWork {
			fields, err := continuousFields(goal, n.Type, s.AllowedFields, n.Props)
			if err != nil {
				return nil, fmt.Errorf("record %s: %w", n.ID, err)
			}
			id := uuid.NewString()
			u := &unitofwork.UnitOfWork{
				ID:            id,
				GoalID:        "goal-" + id[:8],
				Description:   fmt.Sprintf("%s — fix %s", goal, n.ID),
				InputSubgraph: n.ID, AllowedFields: fields,
				ExpectedOutput: "graph mutation",
				Mode:           "non_deterministic",
			}
			uows = append(uows, u)
		}
		for _, u := range uows {
			if err := s.store.Create(u); err != nil {
				return nil, err
			}
		}
		return uows, nil
	}

	// Fallback when no graph is set
	chosen := s.PlanningPack
	if chosen == nil {
		chosen = &pack.Pack{Name: "general", WorkerRoles: []string{"worker"}}
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
	def, err := s.Registry.Resolve(AgentVersion{Name: spec.Name, Role: spec.Role, ModelTier: spec.ModelTier, Tools: spec.Tools, CreatedBy: "steward"})
	if err != nil {
		return nil, err
	}
	if def.ModelTier != "" && def.ModelTier != "cheap" {
		return nil, fmt.Errorf("unsupported worker model tier %q", def.ModelTier)
	}
	worker := NewLLMWorker(def.Name, def.Role, s.Gateway, s.Graph, s.Rules)
	worker.FocusRule = spec.FocusRule
	worker.Definition = def
	return worker, nil
}

func (s *SmartSteward) SetAutonomyMode(mode string) error {
	return nil
}
