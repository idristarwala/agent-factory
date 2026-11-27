package agents

import (
	"fmt"
	"strings"

	"github.com/idristarwala/agent-factory/internal/graph"
)

// FamilyContext holds sibling entities for consistent family-level fixes.
type FamilyContext struct {
	PrimaryID graph.NodeID
	Siblings  []string
	Note      string
}

// GatherFamily collects siblings that share the same group field value.
// In a full graph implementation this would query; the current version returns a lightweight note
// so workers can apply consistent wording across the family.
func GatherFamily(g graph.Graph, entityID graph.NodeID, groupField string) (*FamilyContext, error) {
	if g == nil || entityID == "" {
		return nil, nil
	}
	n, err := g.GetNode(entityID)
	if err != nil {
		return nil, err
	}
	group, _ := n.Props[groupField].(string)
	if group == "" {
		return &FamilyContext{PrimaryID: entityID, Note: "ungrouped item; no siblings to align"}, nil
	}
	// Lightweight note; full sibling list would come from a graph query in production.
	return &FamilyContext{
		PrimaryID: entityID,
		Note:      fmt.Sprintf("item belongs to group %q; apply consistent wording and attribute values to all siblings in the family", group),
	}, nil
}

// PromptFragment returns a short prompt note for the worker.
func (f *FamilyContext) PromptFragment() string {
	if f == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(f.Note)
	if len(f.Siblings) > 0 {
		b.WriteString("\nSiblings: ")
		b.WriteString(strings.Join(f.Siblings, ", "))
	}
	return b.String()
}
