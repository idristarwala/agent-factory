package rules

import (
	"fmt"
	"strings"
	"time"
)

// Policy is a customer-supplied rule, policy, or decision-tree node.
// Domain-agnostic.
type Policy struct {
	ID          string
	Name        string
	Description string
	Condition   string
	Action      string
	Source      string
	Priority    int
}

// DecisionTree is an ordered list of policies.
type DecisionTree struct {
	Name     string
	Policies []Policy
}

// IngestPolicies turns customer policies into versioned rules.
func IngestPolicies(engine *Engine, policies []Policy, createdBy string) []VersionedRule {
	var out []VersionedRule
	for _, p := range policies {
		desc := p.Description
		if desc == "" {
			desc = p.Name
		}
		if p.Condition != "" {
			desc += " | when: " + p.Condition
		}
		if p.Action != "" {
			desc += " | then: " + p.Action
		}
		r := engine.Propose(VersionedRule{
			Description: desc,
			Condition:   p.Condition,
			Action:      p.Action,
			Source:      p.Source,
			CreatedBy:   createdBy,
			CreatedAt:   time.Now(),
		})
		out = append(out, r)
	}
	return out
}

// ParseSimpleTree parses lines of "if <cond> then <action>".
func ParseSimpleTree(text, source string) []Policy {
	var policies []Policy
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cond, action := splitIfThen(line)
		policies = append(policies, Policy{
			Name:      fmt.Sprintf("rule-%d", i+1),
			Condition: cond,
			Action:    action,
			Source:    source,
			Priority:  i,
		})
	}
	return policies
}

func splitIfThen(line string) (cond, action string) {
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "if ") {
		rest := strings.TrimSpace(line[3:])
		if idx := strings.Index(strings.ToLower(rest), " then "); idx >= 0 {
			return strings.TrimSpace(rest[:idx]), strings.TrimSpace(rest[idx+6:])
		}
		return rest, ""
	}
	return line, ""
}
