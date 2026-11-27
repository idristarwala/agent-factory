package models

import "testing"

type recordingGateway struct{ calls int }

func (r *recordingGateway) Generate(Request) (*Response, error) { r.calls++; return &Response{}, nil }

func TestJudgeRoutingUsesRole(t *testing.T) {
	primary, judge := new(recordingGateway), new(recordingGateway)
	g := roleGateway{primary: primary, judge: judge}
	for _, req := range []Request{{CallerRole: "worker", Tier: TierCheap}, {CallerRole: "steward", Tier: TierFrontier}, {Tier: TierFrontier}, {CallerRole: "judge", Tier: TierFrontier}} {
		if _, err := g.Generate(req); err != nil {
			t.Fatal(err)
		}
	}
	if primary.calls != 3 || judge.calls != 1 {
		t.Fatalf("primary calls %d, judge calls %d", primary.calls, judge.calls)
	}
}
