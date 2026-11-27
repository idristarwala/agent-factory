package llm

import "testing"

func TestTierPolicy(t *testing.T) {
	for _, tc := range []struct {
		role   Role
		tier   Tier
		denied bool
	}{
		{RoleWorker, TierFrontier, true},
		{RoleWorker, TierCouncil, true},
		{RoleJudge, TierFrontier, false},
		{RoleWorker, TierCheap, false},
	} {
		err := enforceTier(ChatRequest{CallerRole: tc.role, Tier: tc.tier})
		if tc.denied {
			if e, ok := err.(*Error); !ok || e.Type != "CAPABILITY" {
				t.Errorf("%s at %s: expected capability error, got %v", tc.role, tc.tier, err)
			}
		} else if err != nil {
			t.Errorf("%s at %s: %v", tc.role, tc.tier, err)
		}
	}
}
