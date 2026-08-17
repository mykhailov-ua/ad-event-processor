// Flow router selection benches (harness: flow_router).
// RCU snapshot is fully in-memory: no Redis, no PG on the read path.
package ingestion

import (
	"testing"

	"github.com/google/uuid"
)

var flowRouterBenchSink FlowSelection

func benchFlowRouter(tb testing.TB) (*FlowRouter, [64][16]byte) {
	tb.Helper()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	offerB := uuid.MustParse("00000000-0000-4000-8000-000000000102")
	snap := &FlowPathSnapshot{
		Paths: []FlowPath{
			{
				Weight: 70,
				Landers: []FlowLanderEntry{
					{LanderID: landerA, Weight: 70, URL: []byte("https://lander-a.test/lp")},
					{LanderID: landerB, Weight: 30, URL: []byte("https://lander-b.test/lp")},
				},
				Offers: []FlowOfferEntry{
					{OfferID: offerA, Weight: 50},
					{OfferID: offerB, Weight: 50},
				},
			},
			{
				Weight: 30,
				Landers: []FlowLanderEntry{
					{LanderID: landerB, Weight: 100, URL: []byte("https://lander-b.test/alt")},
				},
				Offers: []FlowOfferEntry{
					{OfferID: offerB, Weight: 100},
				},
			},
		},
	}
	router := NewFlowRouter()
	router.Publish(snap)

	var users [64][16]byte
	for i := range users {
		users[i][0] = byte('u')
		users[i][15] = byte(i)
	}
	return router, users
}

// BenchmarkFlowRouter_BanditSelect (harness: bandit_router) — B-GMA-M6, < 5 µs, 0 allocs.
func BenchmarkFlowRouter_BanditSelect(b *testing.B) {
	router, users := benchFlowRouter(b)
	b.ReportAllocs()
	b.ResetTimer()
	var sel FlowSelection
	var url []byte
	var ok bool
	for i := 0; i < b.N; i++ {
		sel, url, ok = router.BanditSelect(users[i&63][:])
	}
	flowRouterBenchSink = sel
	if ok {
		flowRouterBenchSink.PathIdx += len(url)
	}
}

// BenchmarkFlowRouter_Select (harness: flow_router) — B-GM-M3, < 2 µs, 0 allocs.
func BenchmarkFlowRouter_Select(b *testing.B) {
	router, users := benchFlowRouter(b)
	b.ReportAllocs()
	b.ResetTimer()
	var sel FlowSelection
	var ok bool
	for i := 0; i < b.N; i++ {
		sel, ok = router.Select(users[i&63][:])
	}
	flowRouterBenchSink = sel
	flowRouterBenchSink.PathIdx += boolToInt(ok)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
