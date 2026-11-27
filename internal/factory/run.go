// Package factory executes records against an explicitly selected factory revision.
package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/artifact"
	"github.com/idristarwala/agent-factory/internal/memory"
	"github.com/idristarwala/agent-factory/internal/tools"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/graph"
	"github.com/idristarwala/agent-factory/internal/models"
	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/rules"
	"github.com/idristarwala/agent-factory/internal/schema"
	"github.com/idristarwala/agent-factory/internal/unitofwork"
)

type Criterion struct {
	Text      string              `json:"text"`
	Citations []artifact.Citation `json:"citations"`
}
type Spec struct {
	BrowserTool     string            `json:"browser_tool,omitempty"`
	ProposeTool     string            `json:"propose_tool,omitempty"`
	Goal            string            `json:"goal,omitempty"`
	Criteria        []Criterion       `json:"criteria,omitempty"`
	ProposalVersion int               `json:"proposal_version,omitempty"`
	Artifacts       []artifact.Ref    `json:"artifacts,omitempty"`
	RecordType      string            `json:"record_type"`
	Fields          map[string]string `json:"fields"`
	Required        []string          `json:"required"`
	Worker          string            `json:"worker"`
}
type Plan struct {
	BrowserTool            *BrowserTool  `json:"browser_tool,omitempty"`
	ProposalTool           *ProposalTool `json:"proposal_tool,omitempty"`
	UseGoalArtifacts       bool          `json:"use_goal_artifacts,omitempty"`
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
	if def.BrowserTool != "" {
		plan.BrowserTool, err = pinBrowserTool(st, name, def.BrowserTool)
		if err != nil {
			return Plan{}, err
		}
	}
	if def.ProposeTool != "" {
		plan.ProposalTool, err = pinProposalTool(st, name, def.ProposeTool)
		if err != nil {
			return Plan{}, err
		}
	}
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
	if p.BrowserTool != nil {
		return nil, agents.Decision{}, fmt.Errorf("browser capture requires a workspace run")
	}
	return p.evaluate(gw, goal, e, nil, artifact.Evidence{}, nil)
}
func (p Plan) evaluate(gw models.Gateway, goal string, e schema.Entity, origin *SourceLineage, goalEvidence artifact.Evidence, toolCalls map[string]int) (*graph.Mutation, agents.Decision, error) {
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
	var media tools.RecordEvidence
	if origin != nil && len(e.ImageURLs) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		var err error
		if toolCalls != nil {
			toolCalls["collect_mapped_images"]++
		}
		media, err = tools.CollectMappedImages(ctx, e.ImageURLs)
		if err != nil {
			return nil, agents.Decision{}, err
		}
	}
	evidence := media.Text
	if tool := p.ProposalTool; tool != nil {
		evidence = append(evidence, fmt.Sprintf("Propose field changes for tool %q revision %d (%s). The judge and human reviewer will review the external change.", tool.Name, tool.ToolRevision, tool.SideEffect))
	}
	for _, criterion := range p.Spec.Criteria {
		evidence = append(evidence, "Approved acceptance criterion: "+criterion.Text)
	}
	if goalEvidence.Text != "" {
		evidence = append(evidence, goalEvidence.Text)
		// Goal evidence numbers its images from one.
		media.Images = append(slices.Clone(goalEvidence.Images), media.Images...)
	}
	if len(e.Relations) > 0 {
		data, err := json.Marshal(e.Relations)
		if err != nil {
			return nil, agents.Decision{}, err
		}
		evidence = append(evidence, "Record relationships: "+string(data))
	}
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
	mutation, err := worker.Execute(&unitofwork.UnitOfWork{ID: e.ID, InputSubgraph: graph.NodeID(e.ID), Evidence: evidence, Images: media.Images, Description: goal, AllowedFields: fields, RequiredFields: p.Spec.Required, FieldTypes: p.Spec.Fields, PreserveStatus: true})
	if err != nil {
		return nil, agents.Decision{}, err
	}
	if tool := p.ProposalTool; tool != nil {
		mutation.ToolProposal = &graph.ToolProposal{
			Name: tool.Name, ToolRevision: tool.ToolRevision, Pack: p.Factory, PackRevision: tool.PackRevision, SideEffect: tool.SideEffect,
			Arguments: map[string]any{tool.RecordArgument: e.ID, tool.ChangesArgument: mutation.Updates[0].Props},
		}
	}
	mutation.Evidence = append(mutation.Evidence, "Requested goal: "+goal)
	mutation.Evidence = append(mutation.Evidence, evidence...)
	judge := agents.NewLLMJudge(gw, g)
	judge.Definition = p.Judge
	d, err := judge.Evaluate(mutation, active)
	return mutation, d, err
}
