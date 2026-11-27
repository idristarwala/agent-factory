package learning

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"

	"github.com/idristarwala/agent-factory/internal/agents"
	"github.com/idristarwala/agent-factory/internal/revision"
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
	RuleEngine             *rules.Engine
	Registry               *agents.Registry
	SpecialistAfter        int
	ReplayGate             bool
	ReplayRequiresApproval bool
	Replay                 func(rules.VersionedRule, []rules.VersionedRule) (*rules.ReplayReport, error)
	cumulative             map[string]int
}

func NewLearningLoop(re *rules.Engine, reg *agents.Registry) *LearningLoop {
	return &LearningLoop{
		RuleEngine:      re,
		Registry:        reg,
		SpecialistAfter: 5,
		cumulative:      make(map[string]int),
	}
}

func (l *LearningLoop) Review(decisions []agents.Decision, learnings []string) []Gap {
	var signals []FailureSignal
	for _, msg := range learnings {
		signals = RecordFailure(signals, errors.New(msg))
	}
	return l.ReviewSignals(decisions, signals)
}

func (l *LearningLoop) ReviewSignals(decisions []agents.Decision, signals []FailureSignal) []Gap {
	if l.cumulative == nil {
		l.cumulative = make(map[string]int)
	}
	var gaps []Gap
	rejectReasons := map[string]int{}
	for _, d := range decisions {
		if !d.Approved {
			key := RejectionPattern(d.Reason)
			rejectReasons[key]++
			l.cumulative[key]++
		}
	}
	for theme, count := range rejectReasons {
		total := l.cumulative[theme]
		if total >= 2 {
			gaps = append(gaps, Gap{
				Pattern:     theme,
				Evidence:    []string{fmt.Sprintf("%d this round, %d cumulative", count, total)},
				Occurrences: total,
				Severity:    l.severity(total),
			})
		}
	}
	for _, signal := range signals {
		pattern := "worker failure: " + signal.Cause
		l.cumulative[pattern] += signal.Count
		gaps = append(gaps, Gap{Pattern: pattern, Evidence: []string{BuildActionableRule(signal)}, Occurrences: l.cumulative[pattern], Severity: l.severity(l.cumulative[pattern])})
	}
	return gaps
}

func (l *LearningLoop) ProposeUpdates(gaps []Gap) []AgentUpdate {
	var updates []AgentUpdate
	for _, g := range gaps {
		actionable := ActionableRule(g.Pattern, g.Evidence)
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

// ActionableRule is the exported form used by the learning path.
func ActionableRule(pattern string, evidence []string) string {
	return actionableRule(pattern, evidence)
}

func actionableRule(pattern string, evidence []string) string {
	if strings.HasPrefix(pattern, "worker failure: ") && len(evidence) > 0 {
		return evidence[0]
	}
	lower := strings.ToLower(pattern)
	switch {
	case strings.Contains(lower, "title"):
		return "When correcting a title, preserve the source meaning and verify changes against the evidence."
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

func (l *LearningLoop) Apply(updates []AgentUpdate) ([]string, error) {
	var applied []string
	for _, u := range updates {
		switch u.ChangeType {
		case "rule":
			active := !l.ReplayGate
			id := ruleID(u.Payload["pattern"])
			r, err := l.RuleEngine.Save(rules.VersionedRule{
				ID:          id,
				Description: u.Payload["description"],
				Source:      u.Payload["source"],
				CreatedBy:   "steward",
				CreatedAt:   time.Now(),
				Active:      active,
			}, 0)
			if errors.Is(err, revision.ErrConflict) {
				if !l.ReplayGate || l.Replay == nil {
					continue
				}
				history, readErr := l.RuleEngine.List(id)
				if readErr != nil {
					return applied, readErr
				}
				if len(history) != 1 || history[0].CreatedBy != "steward" || history[0].Active || history[0].Description != u.Payload["description"] {
					continue
				}
				r, err = history[0], nil
			}
			if err != nil {
				return applied, err
			}
			if l.ReplayGate && l.Replay != nil {
				baseline, err := l.RuleEngine.Current()
				if err != nil {
					return applied, err
				}
				report, err := l.Replay(r, baseline)
				if err != nil {
					return applied, fmt.Errorf("rule replay: %w", err)
				}
				active = ShouldActivate(report) && !l.ReplayRequiresApproval
				r.Replay = report
				r.Active = active
				if active {
					r.ActivatedFrom = r.Version
				}
				r, err = l.RuleEngine.Save(r, r.Version)
				if errors.Is(err, revision.ErrConflict) {
					continue
				}
				if err != nil {
					return applied, err
				}
			}
			status := "active"
			if !active {
				status = "proposed (awaiting replay)"
				if r.Replay != nil {
					status = "proposed (replay did not improve)"
					if ShouldActivate(r.Replay) {
						status = "proposed (awaiting human approval)"
					}
				}
			}
			applied = append(applied, fmt.Sprintf("rule %s v%d [%s]: %s", r.ID[:8], r.Version, status, r.Description))
		case "new_agent":
			base, err := l.Registry.Resolve(agents.AgentVersion{Name: "generic", Role: "worker", ModelTier: "cheap", CreatedBy: "steward"})
			if err != nil {
				return applied, err
			}
			_, err = l.Registry.Save(agents.AgentVersion{
				Name:   u.Payload["role"],
				Config: base.Config, Tools: base.Tools,
				CreatedBy: "steward",
				Role:      "worker",
				Version:   u.Payload["version"],
				ModelTier: "cheap",
				StartedAt: time.Now(),
			}, 0)
			if errors.Is(err, revision.ErrConflict) {
				continue
			}
			if err != nil {
				return applied, err
			}
			applied = append(applied, fmt.Sprintf("registered agent %s", u.Payload["role"]))
		}
	}
	return applied, nil
}

func RejectionPattern(reason string) string {
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

func (l *LearningLoop) severity(count int) string {
	if l.SpecialistAfter > 0 && count >= l.SpecialistAfter {
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

func ruleID(pattern string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("steward-rule/"+pattern)).String()
}

// WorkerForPattern dispatches only when the governing rule is active this round.
func (l *LearningLoop) WorkerForPattern(pattern string, active []agents.Rule) (agents.WorkerSpec, error) {
	spec := agents.WorkerSpec{Name: "generic", Role: "worker", ModelTier: "cheap"}
	if pattern == "" {
		return spec, nil
	}
	id := ruleID(pattern)
	for _, r := range active {
		if r.ID != id {
			continue
		}
		name := "specialist-" + slug(pattern)
		history, err := l.Registry.History(name)
		if err != nil {
			return spec, err
		}
		if len(history) > 0 {
			def := history[len(history)-1]
			if !def.Retired && def.Role == "worker" {
				spec.Name = name
				spec.FocusRule = id
			}
		}
		break
	}
	return spec, nil
}
