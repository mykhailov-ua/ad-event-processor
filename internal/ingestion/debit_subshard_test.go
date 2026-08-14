package ingestion

import (
	"strconv"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDebitShard_highVolumeSpread(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	f := &UnifiedFilter{
		sharder: sharder,
		rdbs:    make([]redis.UniversalClient, 4),
	}
	campID := uuid.New()
	camp := &domain.Campaign{
		ID:            campID,
		BehaviorFlags: domain.BehaviorHighVolumeDebit,
	}

	seen := make(map[int]struct{})
	for i := range 64 {
		userID := "user-" + strconv.Itoa(i)
		shard, sub, err := f.resolveDebitShard(campID, userID, "", camp)
		require.NoError(t, err)
		require.GreaterOrEqual(t, sub, 0)
		require.Less(t, sub, domain.HighVolumeDebitSubShards)
		seen[shard] = struct{}{}
	}
	assert.GreaterOrEqual(t, len(seen), 2, "high-volume debits should spread across shards")
}

func TestDebitSubSlot_stable(t *testing.T) {
	camp := &domain.Campaign{
		ID:            uuid.New(),
		BehaviorFlags: domain.BehaviorHighVolumeDebit,
	}
	a := debitSubSlot(camp, "user-42", "")
	b := debitSubSlot(camp, "user-42", "")
	assert.Equal(t, a, b)
	assert.NotEqual(t, debitSubSlot(camp, "user-99", ""), a)
}

func TestBudgetQuotaKeySub_hashTag(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	key := domain.BudgetQuotaKeySub(id, 2)
	assert.Contains(t, key, "{550e8400-e29b-41d4-a716-446655440000:slot_2}")
	assert.Contains(t, key, "budget:quota:")
}
