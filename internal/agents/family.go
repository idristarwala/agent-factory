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
		return &FamilyContext{PrimaryID: entityID, Note: "ungrouped item"}, nil
	}
	// In a real graph we would query by group; here we return a lightweight note.
	return &FamilyContext{
		PrimaryID: entityID,
		Note:      fmt.Sprintf("item belongs to group %q; apply consistent wording to siblings", group),
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
