package pack

// Release pins the validated pack and its CI evidence to one Git commit.
type Release struct {
	ProposalRevision int    `json:"proposal_revision,omitempty"`
	ReviewNote       string `json:"review_note,omitempty"`
	Status           string `json:"status"`
	Commit           string `json:"commit"`
	PackSHA          string `json:"pack_sha256"`
	EvidenceSHA      string `json:"evidence_sha256"`
	Pack             []byte `json:"pack_bytes"`
	Evidence         []byte `json:"evidence_bytes"`
}
