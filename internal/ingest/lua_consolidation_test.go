package ingest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func edgeSlotPick(campaignID uuid.UUID, table *slotTable) (int, bool) {
	if table == nil {
		return 0, false
	}
	slot := CRC32Castagnoli(&campaignID) & 1023
	return int(table[slot]), true
}

func TestFault_EdgeSlotMapParity(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	table := buildSlotTable(4)
	sharder.SwapSnapshot(7, table, 0)

	const n = 4096
	mismatches := 0
	for range n {
		id := uuid.New()
		goShard := sharder.GetShard(id)
		edgeShard, ok := edgeSlotPick(id, table)
		require.True(t, ok)
		if goShard != edgeShard {
			mismatches++
		}
	}
	require.Equal(t, 0, mismatches, "edge slot map must match StaticSlotSharder")

	faultproof.Log(t, "edge_slot_map_parity", map[string]string{
		"samples":    fmt.Sprintf("%d", n),
		"mismatches": fmt.Sprintf("%d", mismatches),
		"version":    fmt.Sprintf("%d", sharder.SnapshotVersion()),
	})
}

func TestUnifiedFilter_LuaConsolidatedPrechecks(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := attachFilterDeadline(t.Context(), time.Second)
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	reg := &mockRegistry{}
	f := newRealRedisUnifiedFilter(t, redisClient)
	f.registry = reg
	f.SetLuaFastPathEnabled(true)
	f.SetTTCMin(0)
	f.SetRegionCode(0)
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	seedCampaignBudget(t, ctx, redisClient, campID)

	require.NoError(t, redisClient.HSet(ctx, PlacementBlacklistKey(campID), "zone-bad", "1").Err())
	require.NoError(t, redisClient.SAdd(ctx, fraudBlacklistKey, "203.0.113.66").Err())

	placementEvt := &domain.Event{
		Type:        "impression",
		CampaignID:  campID,
		ClickID:     uuid.NewString(),
		IP:          "203.0.113.1",
		PlacementID: "zone-bad",
	}
	require.ErrorIs(t, f.Check(ctx, placementEvt), ErrPlacementBlocked)

	fraudEvt := &domain.Event{
		Type:       "impression",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
		IP:         "203.0.113.66",
	}
	require.NoError(t, f.Check(ctx, fraudEvt))

	quotaReg := &entitlementsTestRegistry{
		CampaignRegistry: reg,
		maxRPD:           1,
	}
	f.registry = quotaReg
	quotaEvt := &domain.Event{
		Type:       "impression",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
		IP:         "203.0.113.77",
	}
	require.NoError(t, f.Check(ctx, quotaEvt))
	quotaEvt.ClickID = uuid.NewString()
	require.ErrorIs(t, f.Check(ctx, quotaEvt), ErrDailyQuotaExceeded)
}

type entitlementsTestRegistry struct {
	domain.CampaignRegistry
	maxRPD uint64
}

func (r *entitlementsTestRegistry) GetEntitlements(customerID uuid.UUID) (licensing.Entitlements, bool) {
	return licensing.Entitlements{
		Limits: licensing.Limits{MaxRequestsPerDay: r.maxRPD},
	}, true
}

func (r *entitlementsTestRegistry) GetCampaign(uuid.UUID) (*domain.Campaign, bool) {
	return nil, false
}

func TestUnifiedFilter_NoIPRateLimitKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := attachFilterDeadline(t.Context(), time.Second)
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	f := newRealRedisUnifiedFilter(t, redisClient)
	f.SetLuaFastPathEnabled(false)
	f.SetTTCMin(0)
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	seedCampaignBudget(t, ctx, redisClient, campID)

	var rlKey []byte
	rlKey = appendCampaignHashTag(rlKey[:0], campID)
	rlKey = append(rlKey, "rl:ip:203.0.113.50"...)
	require.NoError(t, redisClient.Set(ctx, string(rlKey), 0, 0).Err())

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.50",
		UserID:     "u-rl",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	require.NoError(t, f.Check(ctx, evt))

	val, err := redisClient.Get(ctx, string(rlKey)).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(0), val, "rl:ip key must not be incremented by Lua")
}

func TestUnifiedFilter_TierDegradationNearDeadline(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	reg := &benchWorstRegistry{}
	f := newRealRedisUnifiedFilter(t, redisClient)
	f.registry = reg
	f.SetLuaFastPathEnabled(false)
	f.SetTTCMin(500 * time.Millisecond)
	require.NoError(t, f.PreloadScripts(ctx))

	campID := uuid.New()
	camp, ok := reg.GetCampaign(campID)
	require.True(t, ok)
	require.NoError(t, redisClient.Set(ctx, camp.BudgetCampaignKey, 9_000_000_000_000_000, 0).Err())
	require.NoError(t, redisClient.Set(ctx, camp.FcapKeyPrefix+"degrade-user", 999, 0).Err())

	before := testutil.ToFloat64(metrics.FilterTierDegradedTotal)

	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.88",
		UserID:     "degrade-user",
		CampaignID: campID,
		ClickID:    uuid.NewString(),
	}
	evt.FilterDeadlineMono = monotonicNano() + 500_000

	require.NoError(t, f.Check(ctx, evt))
	after := testutil.ToFloat64(metrics.FilterTierDegradedTotal)
	require.Greater(t, after-before, 0.0)
}
