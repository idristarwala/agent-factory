package models

// ResponseSchema holds an optional JSON schema for provider-enforced structured output.
// Addresses issue #14.
type ResponseSchema struct {
	Name   string
	Schema map[string]any
	Strict bool
}
