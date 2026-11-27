// Package schemas contains the published input contracts used by the CLI.
package schemas

import _ "embed"

//go:embed pack.schema.json
var Pack string

//go:embed rule.schema.json
var Rule string
