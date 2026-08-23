package ingestion

import (
	"context"
	"strconv"
	"testing"
	"time"

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

func TestFcapKeyPrefixForDebit_subShard(t *testing.T) {
	camp := &domain.Campaign{
		ID:            uuid.New(),
		BehaviorFlags: domain.BehaviorHighVolumeDebit,
	}
	prefix := fcapKeyPrefixForDebit(camp, "user-42", "")
	assert.Contains(t, prefix, ":slot_")
	assert.Contains(t, prefix, "fcap:c:")
	assert.Equal(t, domain.FcapKeyPrefix(camp.ID, ""), fcapKeyPrefixForDebit(camp, "", ""))
}

func TestDebitSubShard_plainCampaignSingleHashTag_holdout(t *testing.T) {
	camp := &domain.Campaign{ID: uuid.New()}
	require.Equal(t, 0, camp.DebitSubShardCount())
	require.Equal(t, 0, debitSubSlot(camp, "user-1", "click-1"))

	plainKey := domain.BudgetQuotaKey(camp.ID)
	require.NotContains(t, plainKey, ":slot_")

	sharder := NewStaticSlotSharder(4)
	f := &UnifiedFilter{
		sharder: sharder,
		rdbs:    make([]redis.UniversalClient, 4),
	}
	shardA, subA, err := f.resolveDebitShard(camp.ID, "user-1", "", camp)
	require.NoError(t, err)
	shardB, subB, err := f.resolveDebitShard(camp.ID, "user-99", "click-99", camp)
	require.NoError(t, err)
	require.Equal(t, 0, subA)
	require.Equal(t, 0, subB)
	require.Equal(t, shardA, shardB)
	require.Equal(t, sharder.GetShard(camp.ID), shardA)
}

func TestHighVolumeDebit_subShardQuotaKeysDistinct(t *testing.T) {
	id := uuid.New()
	keys := make(map[string]struct{}, domain.HighVolumeDebitSubShards)
	for sub := range domain.HighVolumeDebitSubShards {
		keys[domain.BudgetQuotaKeySub(id, sub)] = struct{}{}
	}
	require.Len(t, keys, domain.HighVolumeDebitSubShards)
}

func TestUnifiedFilter_highVolumeDebit_debitsSubShardQuotaKey(t *testing.T) {
	ctx := context.Background()
	mr, cleanup := setupMiniredis(t)
	defer cleanup()

	campID := uuid.New()
	camp := &domain.Campaign{
		ID:            campID,
		CustomerID:    uuid.New(),
		PacingMode:    domain.PacingModeAsap,
		BehaviorFlags: domain.BehaviorHighVolumeDebit,
	}
	reg := benchRegistryForCampaign(camp)
	f := newQuotaUnifiedFilter(t, mr)
	f.registry = reg
	f.SetLuaFastPathEnabled(true)
	f.SetTTCMin(0)
	require.NoError(t, f.PreloadScripts(ctx))

	var userA, userB string
	var subA, subB int
	for i := range 64 {
		userID := "hv-user-" + strconv.Itoa(i)
		sub := debitSubSlot(camp, userID, "")
		if userA == "" {
			userA, subA = userID, sub
			continue
		}
		if sub != subA {
			userB, subB = userID, sub
			break
		}
	}
	require.NotEmpty(t, userB)

	keyA := domain.BudgetQuotaKeySub(campID, subA)
	keyB := domain.BudgetQuotaKeySub(campID, subB)
	const seedMicro = int64(10_000_000)
	require.NoError(t, mr.Set(ctx, keyA, seedMicro, 0).Err())
	require.NoError(t, mr.Set(ctx, keyB, seedMicro, 0).Err())

	evt := &domain.Event{
		Type:       "impression",
		CampaignID: campID,
		UserID:     userA,
		ClickID:    uuid.NewString(),
		IP:         "203.0.113.60",
	}
	checkCtx := attachFilterDeadline(ctx, time.Second)
	require.NoError(t, f.Check(checkCtx, evt))

	afterA, err := mr.Get(ctx, keyA).Int64()
	require.NoError(t, err)
	afterB, err := mr.Get(ctx, keyB).Int64()
	require.NoError(t, err)
	require.Less(t, afterA, seedMicro, "debit must hit sub-slot %d key %s", subA, keyA)
	require.Equal(t, seedMicro, afterB, "other sub-slot %d key must be untouched", subB)
	_ = subB
}
