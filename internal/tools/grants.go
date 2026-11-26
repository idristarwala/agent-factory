package tools

// Grant defines which packs may use which tools and side-effect classes.
// Starting point for the access-as-config model in issue #9.
type Grant struct {
	Pack        string   `json:"pack"`
	Tools       []string `json:"tools"`
	SideEffects []string `json:"side_effects"` // read-only, reversible, irreversible
}

// CheckGrant returns true if the pack is allowed to use the tool with the given side effect.
func CheckGrant(grants []Grant, pack, tool, sideEffect string) bool {
	for _, g := range grants {
		if g.Pack != pack {
			continue
		}
		for _, t := range g.Tools {
			if t == tool || t == "*" {
				for _, se := range g.SideEffects {
					if se == sideEffect || se == "*" {
						return true
					}
				}
			}
		}
	}
	return false
}
