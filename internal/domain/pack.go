package domain

// Pack describes a domain (catalog, pricing, inventory, etc.).
// Steward uses packs to know how to break goals into units of work
// and which worker roles to create.

type Pack struct {
	Name         string
	Description  string
	NodeTypes    []string
	WorkerRoles  []string
	DefaultRules []string
}

func CatalogPack() Pack {
	return Pack{
		Name:        "catalog",
		Description: "Product titles, descriptions, attributes, variants",
		NodeTypes:   []string{"product", "variant", "attribute"},
		WorkerRoles: []string{"title-enricher", "description-enricher", "attribute-filler"},
		DefaultRules: []string{
			"Titles must be descriptive and include key attributes",
			"Descriptions must be factual and customer-friendly",
			"Do not invent attributes that are not supported by evidence",
		},
	}
}

func PricingPack() Pack {
	return Pack{
		Name:        "pricing",
		Description: "Price rules, promotions, margin checks",
		NodeTypes:   []string{"price", "promo", "competitor"},
		WorkerRoles: []string{"price-proposer", "margin-checker"},
		DefaultRules: []string{
			"Never propose a price below cost",
			"Flag large deviations from current price",
		},
	}
}

func All() []Pack {
	return []Pack{CatalogPack(), PricingPack()}
}
