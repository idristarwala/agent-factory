package rules

import (
	"errors"
	"github.com/idristarwala/agent-factory/internal/revision"
)

// Install seeds a stable rule once. Restarting a factory preserves subsequent edits.
func (e *Engine) Install(r VersionedRule) error {
	_, err := e.Save(r, 0)
	if errors.Is(err, revision.ErrConflict) {
		return nil
	}
	return err
}
