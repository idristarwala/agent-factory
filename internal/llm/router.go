package llm

import "fmt"

func enforceTier(req ChatRequest) error {
	if req.CallerRole == RoleWorker && (req.Tier == TierFrontier || req.Tier == TierCouncil) {
		return &Error{
			Type:    "CAPABILITY",
			Message: fmt.Sprintf("worker role cannot request tier %s", req.Tier),
		}
	}
	return nil
}
