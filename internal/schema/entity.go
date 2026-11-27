package schema

// Entity is the single shape everything maps to.
// Catalog items, prices, inventory rows, merchandising placements — all become Entities.
type Entity struct {
	ImageURLs  []string       `json:"image_urls,omitempty"`
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Domain     string         `json:"domain"`
	Attributes map[string]any `json:"attributes"`
	Relations  []Relation     `json:"relations,omitempty"`
	Version    int            `json:"version"`
}

// Relation links entities (variant-of, located-at, etc.).
type Relation struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

// Decision is what the customer sees.
type Decision struct {
	EntityID string         `json:"entity_id"`
	Before   map[string]any `json:"before"`
	After    map[string]any `json:"after"`
	Reason   string         `json:"reason"`
	Status   string         `json:"status"`
	Feedback string         `json:"feedback,omitempty"`
}

// StoreMode is chosen by the Steward, never by the user.
type StoreMode string

const (
	ModeMemory StoreMode = "memory"
	ModeDB     StoreMode = "db"
)

// ChooseMode lets the Steward pick storage based on volume.
// Business users never see this.
func ChooseMode(entityCount int, explicit string) StoreMode {
	if explicit == "db" || explicit == "memory" {
		return StoreMode(explicit)
	}
	if entityCount > 10_000 {
		return ModeDB
	}
	return ModeMemory
}
