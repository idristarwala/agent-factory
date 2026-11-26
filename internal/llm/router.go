package llm

import (
	"context"
	"fmt"
)

// Router implements Client and enforces tier policy.
// Workers may only use TierCheap. Steward and Judge may use any tier.
type Router struct {
	client Client
}

func NewRouter(c Client) *Router {
	return &Router{client: c}
}

func (r *Router) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if err := enforceTier(req); err != nil {
		return nil, err
	}
	return r.client.Chat(ctx, req)
}

func (r *Router) Stream(ctx context.Context, req ChatRequest) (Stream, error) {
	if err := enforceTier(req); err != nil {
		return nil, err
	}
	return r.client.Stream(ctx, req)
}

func enforceTier(req ChatRequest) error {
	if req.CallerRole == RoleWorker && (req.Tier == TierFrontier || req.Tier == TierCouncil) {
		return &Error{
			Type:    "CAPABILITY",
			Message: fmt.Sprintf("worker role cannot request tier %s", req.Tier),
		}
	}
	return nil
}
