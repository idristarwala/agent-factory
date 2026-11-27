package swarm

import "time"

// Budgets are hard limits for a continuous run.
type Budgets struct {
	MaxRounds    int
	MaxWallClock time.Duration
	MaxTokens    int
	Started      time.Time
	TokensUsed   int
}

func NewBudgets(maxRounds int, wall time.Duration, maxTokens int) *Budgets {
	return &Budgets{
		MaxRounds:    maxRounds,
		MaxWallClock: wall,
		MaxTokens:    maxTokens,
		Started:      time.Now(),
	}
}

// Exhausted returns the reason a budget was hit, or empty if still within limits.
func (b *Budgets) Exhausted(round int) string {
	if b.MaxRounds > 0 && round > b.MaxRounds {
		return "max rounds"
	}
	if b.MaxWallClock > 0 && time.Since(b.Started) > b.MaxWallClock {
		return "wall clock"
	}
	if b.MaxTokens > 0 && b.TokensUsed > b.MaxTokens {
		return "token budget"
	}
	return ""
}

func (b *Budgets) AddTokens(n int) {
	b.TokensUsed += n
}
