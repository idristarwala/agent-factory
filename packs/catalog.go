// Package packs provides the bundled catalog example for its standalone demo.
package packs

import _ "embed"

//go:embed catalog/pack.yaml
var CatalogYAML string
