package factory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/pack"
	"github.com/idristarwala/agent-factory/internal/revision"
)

type PackRef struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type ApprovedPack struct {
	ProposalArguments map[string]json.RawMessage `json:"proposal_arguments,omitempty"`
	PackRef
	Commit      string        `json:"commit"`
	PackSHA     string        `json:"pack_sha256"`
	EvidenceSHA string        `json:"evidence_sha256"`
	Rules       []agents.Rule `json:"rules,omitempty"`
}

func pinApprovedPack(st revision.Store, def Spec) (*ApprovedPack, error) {
	ref := *def.PackRelease
	if strings.TrimSpace(ref.Name) == "" || strings.ContainsAny(ref.Name, "/\\") || ref.Version < 1 {
		return nil, fmt.Errorf("pack_release requires a pack name and positive approved revision")
	}
	history, err := revision.Documents(st, "pack-release/"+ref.Name)
	if err != nil {
		return nil, err
	}
	if ref.Version > len(history) {
		return nil, fmt.Errorf("unknown pack release revision")
	}
	var r pack.Release
	if err := json.Unmarshal(history[ref.Version-1].Data, &r); err != nil {
		return nil, err
	}
	if r.Status != "active" || r.ProposalRevision != ref.Version-1 || r.ProposalRevision < 1 {
		return nil, fmt.Errorf("pack release revision must be an approval")
	}
	if fmt.Sprintf("%x", sha256.Sum256(r.Pack)) != r.PackSHA || fmt.Sprintf("%x", sha256.Sum256(r.Evidence)) != r.EvidenceSHA {
		return nil, fmt.Errorf("pack release digest mismatch")
	}
	var receipt struct {
		Commit  string
		Status  string
		PackSHA string `json:"pack_sha256"`
	}
	if err := json.Unmarshal(r.Evidence, &receipt); err != nil || receipt.Status != "passed" || r.Commit == "" || receipt.Commit != r.Commit || receipt.PackSHA != r.PackSHA {
		return nil, fmt.Errorf("pack release evidence does not match the reviewed commit and pack")
	}
	format := "pack.yaml"
	if json.Valid(r.Pack) {
		format = "pack.json"
	}
	p, err := pack.Parse(format, r.Pack)
	if err != nil {
		return nil, err
	}
	if p.Name != ref.Name || !slices.Contains(p.WorkerRoles, def.Worker) || (len(p.NodeTypes) > 0 && !slices.Contains(p.NodeTypes, def.RecordType)) {
		return nil, fmt.Errorf("factory worker or record type is incompatible with reviewed pack %q", ref.Name)
	}
	for _, name := range []string{def.BrowserTool, def.ProposeTool} {
		if name != "" && !slices.Contains(p.ToolsAllowed, name) {
			return nil, fmt.Errorf("tool %q is not allowed by reviewed pack", name)
		}
	}
	pinned := &ApprovedPack{ProposalArguments: p.ProposalArguments[def.ProposeTool], PackRef: ref, Commit: r.Commit, PackSHA: r.PackSHA, EvidenceSHA: r.EvidenceSHA}
	seen := map[string]bool{}
	for _, rule := range p.Rules {
		if seen[rule.ID] {
			return nil, fmt.Errorf("duplicate rule %q in reviewed pack", rule.ID)
		}
		seen[rule.ID] = true
		if rule.Active {
			pinned.Rules = append(pinned.Rules, agents.Rule{ID: rule.ID, Version: ref.Version, Description: rule.Text, Source: "pack-release/" + ref.Name})
		}
	}
	return pinned, nil
}
