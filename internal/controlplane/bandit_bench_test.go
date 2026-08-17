package controlplane

import (
	"math/rand"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/bandit"

	"github.com/google/uuid"
)

// Bandit update benches (harness: bandit_cold_worker).
func BenchmarkBandit_Update(b *testing.B) {
	a := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	bID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	c := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	arms := map[uuid.UUID]bandit.ArmStat{
		a:   {Clicks: 1000, Conversions: 80},
		bID: {Clicks: 1200, Conversions: 40},
		c:   {Clicks: 900, Conversions: 55},
	}
	rng := rand.New(rand.NewSource(99))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bandit.ThompsonWeights(arms, rng)
	}
}
