package agents

import (
	"errors"
	"fmt"
	"github.com/idristarwala/agent-factory/internal/revision"
)

// Resolve returns the configured definition at launch. Defaults are installed
// once so creating another worker cannot undo an operator's edit or retirement.
func (r *Registry) Resolve(initial AgentVersion) (AgentVersion, error) {
	for {
		h, err := r.History(initial.Name)
		if err != nil {
			return AgentVersion{}, err
		}
		if len(h) > 0 {
			v := h[len(h)-1]
			if v.Retired {
				return AgentVersion{}, fmt.Errorf("agent %s is retired at revision %d", v.Name, v.Revision)
			}
			return v, nil
		}
		v, err := r.Save(initial, 0)
		if errors.Is(err, revision.ErrConflict) {
			continue
		}
		return v, err
	}
}
