package swarm

import (
	"github.com/idristarwala/agent-factory/internal/tools"
)

// GrantsEnforcer holds the current grants and can be consulted by the runner.
type GrantsEnforcer struct {
	Grants []tools.Grant
}

func (g *GrantsEnforcer) Allow(pack, tool, sideEffect string) bool {
	return tools.CheckGrant(g.Grants, pack, tool, sideEffect)
}
