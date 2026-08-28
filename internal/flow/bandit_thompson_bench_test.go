package flow

import (
	"math/rand"
	"testing"

	"github.com/google/uuid"
)

func BenchmarkBanditThompsonWeights(b *testing.B) {
	a := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	bID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	c := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	arms := map[uuid.UUID]ArmStat{
		a:   {Clicks: 1000, Conversions: 80},
		bID: {Clicks: 1200, Conversions: 40},
		c:   {Clicks: 900, Conversions: 55},
	}
	rng := rand.New(rand.NewSource(99))
	b.ReportAllocs()
	for b.Loop() {
		ThompsonWeights(arms, rng)
	}
}
