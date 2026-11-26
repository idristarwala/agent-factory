package domain

func InventoryPack() Pack {
	return Pack{
		Name:        "inventory",
		Description: "Stock levels, replenishment, allocation",
		NodeTypes:   []string{"sku", "location", "stock"},
		WorkerRoles: []string{"replenishment-planner", "allocation-optimizer"},
		DefaultRules: []string{
			"Do not allocate more than available stock",
			"Flag SKUs below safety stock",
		},
	}
}

func MerchandisingPack() Pack {
	return Pack{
		Name:        "merchandising",
		Description: "Assortment, placement, visual merchandising",
		NodeTypes:   []string{"assortment", "placement", "display"},
		WorkerRoles: []string{"assortment-planner", "placement-optimizer"},
		DefaultRules: []string{
			"Keep core assortment in every store",
			"Flag seasonal items outside their window",
		},
	}
}
