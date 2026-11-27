package learning

import (
	"fmt"
	"strings"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/rules"
)

// Gap is something the Steward observed that agents are not handling well.
type Gap struct {
	Pattern     string
	Evidence    []string
	Occurrences int
	Severity    string
}

// AgentUpdate is a proposed change to an agent or a new specialized agent.
type AgentUpdate struct {
	TargetAgent string
	ChangeType  string
	Description string
	Payload     map[string]string
}

// LearningLoop is the Steward's self-learning mechanism.
type LearningLoop struct {
	RuleEngine *rules.Engine
	Registry   *agents.Registry
	ReplayGate bool // if true, rules are proposed but not activated until replay passes
}

func NewLearningLoop(re *rules.Engine, reg *agents.Registry) *LearningLoop {
	return &LearningLoop{RuleEngine: re, Registry: reg}
}

func (l *LearningLoop) Review(decisions []agents.Decision, learnings []string) []Gap {
	var gaps []Gap
	rejectReasons := map[string]int{}
	for _, d := range decisions {
		if !d.Approved {
			key := extractTheme(d.Reason)
			rejectReasons[key]++
		}
	}
	for theme, count := range rejectReasons {
		if count >= 2 {
			gaps = append(gaps, Gap{
				Pattern:     theme,
				Evidence:    []string{fmt.Sprintf("%d rejections", count)},
				Occurrences: count,
				Severity:    severity(count),
			})
		}
	}
	for _, learning := range learnings {
		lower := strings.ToLower(learning)
		if strings.Contains(lower, "missing") || strings.Contains(lower, "gap") ||
			strings.Contains(lower, "format") || strings.Contains(lower, "truncat") ||
			strings.Contains(lower, "json") || strings.Contains(lower, "validation") {
			gaps = append(gaps, Gap{
				Pattern:     learning,
				Evidence:    []string{learning},
				Occurrences: 1,
				Severity:    "medium",
			})
		}
	}
	return gaps
}

func (l *LearningLoop) ProposeUpdates(gaps []Gap) []AgentUpdate {
	var updates []AgentUpdate
	for _, g := range gaps {
		// Generate actionable rule content, not just a theme name
		actionable := actionableRule(g.Pattern, g.Evidence)
		updates = append(updates, AgentUpdate{
			TargetAgent: "rules",
			ChangeType:  "rule",
			Description: actionable,
			Payload: map[string]string{
				"description": actionable,
				"source":      "steward-learning",
				"pattern":     g.Pattern,
			},
		})
		if g.Severity == "high" {
			updates = append(updates, AgentUpdate{
				TargetAgent: "new",
				ChangeType:  "new_agent",
				Description: fmt.Sprintf("Create specialist agent for: %s", g.Pattern),
				Payload: map[string]string{
					"role":    "specialist-" + slug(g.Pattern),
					"version": "0.1.0",
				},
			})
		}
	}
	return updates
}

// actionableRule turns a pattern into an instruction the worker can follow.
func actionableRule(pattern string, evidence []string) string {
	lower := strings.ToLower(pattern)
	switch {
	case strings.Contains(lower, "title"):
		return "When correcting titles, keep the original product name and only fix obvious errors. Do not invent attributes."
	case strings.Contains(lower, "description"):
		return "When correcting descriptions, preserve factual claims from the original. Do not add marketing language."
	case strings.Contains(lower, "format"), strings.Contains(lower, "json"), strings.Contains(lower, "validation"):
		return "Return ONLY valid JSON with the changed fields. Do not wrap in markdown. Do not echo the entire entity."
	case strings.Contains(lower, "truncat"), strings.Contains(lower, "token"):
		return "Keep the response short. Return only the fields that changed. Avoid repeating the full entity."
	case strings.Contains(lower, "brand"):
		return "If brand is missing, leave it empty rather than guessing. Only fill brand when evidence is present."
	case strings.Contains(lower, "attribute"):
		return "Only fill attributes that are present in the evidence. Do not invent values."
	default:
		if len(evidence) > 0 {
			return fmt.Sprintf("Address this pattern: %s. Evidence: %s", pattern, evidence[0])
		}
		return fmt.Sprintf("Address this pattern: %s", pattern)
	}
}

func (l *LearningLoop) Apply(updates []AgentUpdate) []string {
	var applied []string
	for _, u := range updates {
		switch u.ChangeType {
		case "rule":
			r := l.RuleEngine.Propose(rules.VersionedRule{
				Description: u.Payload["description"],
				Source:      u.Payload["source"],
				CreatedBy:   "steward",
				CreatedAt:   time.Now(),
				Active:      !l.ReplayGate, // if replay gate is on, propose but don't activate
			})
			status := "active"
			if l.ReplayGate {
				status = "proposed (awaiting replay)"
			}
			applied = append(applied, fmt.Sprintf("rule %s v%d [%s]: %s", r.ID[:8], r.Version, status, r.Description))
		case "new_agent":
			l.Registry.Register(agents.AgentVersion{
				Name:      u.Payload["role"],
				Role:      "worker",
				Version:   u.Payload["version"],
				ModelTier: "cheap",
				StartedAt: time.Now(),
			})
			applied = append(applied, fmt.Sprintf("registered agent %s", u.Payload["role"]))
		}
	}
	return applied
}

func extractTheme(reason string) string {
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "title") {
		return "title quality"
	}
	if strings.Contains(lower, "description") {
		return "description quality"
	}
	if strings.Contains(lower, "attribute") || strings.Contains(lower, "material") {
		return "missing attributes"
	}
	if strings.Contains(lower, "brand") {
		return "brand missing"
	}
	if strings.Contains(lower, "format") || strings.Contains(lower, "json") {
		return "format failure"
	}
	if strings.Contains(lower, "truncat") || strings.Contains(lower, "token") {
		return "truncation"
	}
	return "general rejection"
}

func severity(count int) string {
	if count >= 5 {
		return "high"
	}
	if count >= 2 {
		return "medium"
	}
	return "low"
}

func slug(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}
