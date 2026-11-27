package factory

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/revision"
	"github.com/idristarwala/agent-factory/internal/tools"
)

// ProposalTool pins the approved argument bindings for a record change. Transport
// configuration and credentials are never copied into model or review payloads.
type ProposalTool struct {
	Name            string `json:"name"`
	ToolRevision    int    `json:"tool_revision"`
	PackRevision    int    `json:"pack_revision"`
	SideEffect      string `json:"side_effect"`
	RecordArgument  string `json:"record_argument"`
	ChangesArgument string `json:"changes_argument"`
}

func pinProposalTool(st revision.Store, pack, name string) (*ProposalTool, error) {
	var def struct {
		SideEffect      string `json:"side_effect"`
		RecordArgument  string `json:"record_argument"`
		ChangesArgument string `json:"changes_argument"`
	}
	version, err := latestProposalDefinition(st, "tool/"+name, &def)
	if err != nil {
		return nil, err
	}
	if def.SideEffect != "reversible" && def.SideEffect != "irreversible" {
		return nil, fmt.Errorf("tool %q needs a reversible or irreversible side effect", name)
	}
	if strings.TrimSpace(def.RecordArgument) == "" || strings.TrimSpace(def.ChangesArgument) == "" || def.RecordArgument == def.ChangesArgument {
		return nil, fmt.Errorf("tool %q needs distinct record_argument and changes_argument names", name)
	}
	var policy struct {
		Grants []tools.Grant `json:"grants"`
	}
	packVersion, err := latestProposalDefinition(st, "pack/"+pack, &policy)
	if err != nil {
		return nil, err
	}
	if !tools.CheckGrant(policy.Grants, pack, name, def.SideEffect) {
		return nil, fmt.Errorf("pack %q has no %s grant for %q", pack, def.SideEffect, name)
	}
	return &ProposalTool{Name: name, ToolRevision: version, PackRevision: packVersion, SideEffect: def.SideEffect, RecordArgument: def.RecordArgument, ChangesArgument: def.ChangesArgument}, nil
}

func latestProposalDefinition(st revision.Store, key string, out any) (int, error) {
	docs, err := revision.Documents(st, key)
	if err != nil {
		return 0, err
	}
	if len(docs) == 0 {
		return 0, fmt.Errorf("missing definition %q", key)
	}
	head := docs[len(docs)-1]
	return head.Version, json.Unmarshal(head.Data, out)
}
