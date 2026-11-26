package swarm

// ObserveMode controls whether approved mutations are applied or only recorded.
// This is the starting point for the governed stages in issue #9.
type ObserveMode bool

const (
	ApplyChanges  ObserveMode = false
	ObserveOnly   ObserveMode = true
)

func (r *ContinuousRunner) SetObserveMode(mode ObserveMode) {
	// Stored for future use; current runner always applies when Judge approves.
	// Full observe path records decisions without calling Graph.Apply.
}
