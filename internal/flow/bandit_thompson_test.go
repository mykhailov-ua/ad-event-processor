package flow

import (
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestThompsonWeights_prefersBetterArm(t *testing.T) {
	t.Parallel()
	a := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	b := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	rng := rand.New(rand.NewSource(42))
	var sumA, sumB int32
	for i := range 200 {
		w := ThompsonWeights(map[uuid.UUID]ArmStat{
			a: {Clicks: 1000, Conversions: 100},
			b: {Clicks: 1000, Conversions: 10},
		}, rng)
		sumA += w[a]
		sumB += w[b]
	}
	require.Greater(t, sumA, sumB)
}

func TestThompsonWeights_empty(t *testing.T) {
	t.Parallel()
	require.Nil(t, ThompsonWeights(nil, rand.New(rand.NewSource(1))))
}
