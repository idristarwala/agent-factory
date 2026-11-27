// Package factory executes records against an explicitly selected factory revision.
package factory

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/memory"
	"sort"
	"strings"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/schema"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

type Spec struct {
	RecordType string            `json:"record_type"`
	Fields     map[string]string `json:"fields"`
	Required   []string          `json:"required"`
	Worker     string            `json:"worker"`
}
type Plan struct {
	Memory                 []memory.File `json:"memory,omitempty"`
	Factory                string        `json:"factory"`
	Revision               int           `json:"revision"`
	Spec                   Spec          `json:"spec"`
	Steward, Worker, Judge agents.AgentVersion
	Rules                  []rules.VersionedRule
}

func Prepare(st revision.Store, name string, number int) (Plan, error) {
	if strings.TrimSpace(name) == "" {
		return Plan{}, errors.New("select a factory definition")
	}
	history, err := revision.Documents(st, "factory/"+name)
	if err != nil {
		return Plan{}, err
	}
	if len(history) == 0 || number < 0 || number > len(history) {
		return Plan{}, fmt.Errorf("unknown factory revision for %s", name)
	}
	if number == 0 {
		number = len(history)
	}
	var def Spec
	if err := json.Unmarshal(history[number-1].Data, &def); err != nil {
		return Plan{}, err
	}
	if def.RecordType == "" || len(def.Fields) == 0 {
		return Plan{}, errors.New("factory definition needs a record_type and editable fields")
	}
	for k, t := range def.Fields {
		if strings.TrimSpace(k) == "" {
			return Plan{}, errors.New("empty field name")
		}
		switch t {
		case "string", "integer", "number", "boolean", "object", "array":
		default:
			return Plan{}, fmt.Errorf("unsupported field type %q", t)
		}
	}
	for _, k := range def.Required {
		if _, ok := def.Fields[k]; !ok {
			return Plan{}, fmt.Errorf("required field %s is not editable", k)
		}
	}
	if def.Worker == "" {
		def.Worker = "worker"
	}
	registry := agents.NewStoredRegistry(st)
	plan := Plan{Factory: name, Revision: number, Spec: def}
	for _, item := range []struct {
		key, role, tier, actor string
		dest                   *agents.AgentVersion
	}{{"steward", "steward", "frontier", "system", &plan.Steward}, {def.Worker, "worker", "cheap", "steward", &plan.Worker}, {"judge", "judge", "frontier", "system", &plan.Judge}} {
		v, err := registry.Resolve(agents.AgentVersion{Name: item.key, Role: item.role, ModelTier: item.tier, CreatedBy: item.actor})
		if err != nil {
			return Plan{}, err
		}
		if v.ModelTier != item.tier {
			return Plan{}, fmt.Errorf("unsupported %s tier %q", item.role, v.ModelTier)
		}
		*item.dest = v
	}
	plan.Rules, err = rules.NewStoredEngine(st).Current()
	return plan, err
}
func (p Plan) Evaluate(gw models.Gateway, goal string, e schema.Entity) (*graph.Mutation, agents.Decision, error) {
	return p.evaluate(gw, goal, e, nil)
}
func (p Plan) evaluate(gw models.Gateway, goal string, e schema.Entity, origin *SourceLineage) (*graph.Mutation, agents.Decision, error) {
	if e.ID == "" || e.Type != p.Spec.RecordType {
		return nil, agents.Decision{}, fmt.Errorf("record %q must have type %q", e.ID, p.Spec.RecordType)
	}
	g := graph.NewMemoryGraph()
	g.AddNode(graph.Node{ID: graph.NodeID(e.ID), Type: e.Type, Props: e.Attributes})
	active := make([]agents.Rule, 0, len(p.Rules))
	for _, r := range p.Rules {
		active = append(active, agents.Rule{ID: r.ID, Version: r.Version, Condition: r.Condition, Action: r.Action, Description: r.Description, Source: r.Source})
	}
	worker := agents.NewLLMWorker(p.Worker.Name, p.Worker.Role, gw, g, active)
	worker.Definition = p.Worker
	fields := make([]string, 0, len(p.Spec.Fields))
	for k := range p.Spec.Fields {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	var evidence []string
	if origin != nil {
		data, err := json.Marshal(origin)
		if err != nil {
			return nil, agents.Decision{}, err
		}
		evidence = append(evidence, "Source lineage: "+string(data))
	}
	for _, file := range p.Memory {
		evidence = append(evidence, fmt.Sprintf("Factory memory %q revision %d (%s): %s", file.Path, file.Version, file.Author, file.Content))
	}
	mutation, err := worker.Execute(&unitofwork.UnitOfWork{ID: e.ID, InputSubgraph: graph.NodeID(e.ID), Evidence: evidence, Description: goal, AllowedFields: fields, RequiredFields: p.Spec.Required, FieldTypes: p.Spec.Fields, PreserveStatus: true})
	if err != nil {
		return nil, agents.Decision{}, err
	}
	mutation.Evidence = append(mutation.Evidence, evidence...)
	judge := agents.NewLLMJudge(gw, g)
	judge.Definition = p.Judge
	d, err := judge.Evaluate(mutation, active)
	return mutation, d, err
}
