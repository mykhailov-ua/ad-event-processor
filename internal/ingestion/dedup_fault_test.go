package ingestion

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/bidshard/ad-event-processor/internal/dedup"
	"github.com/bidshard/ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_DedupCrashRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	campaignID := seedFaultCampaign(t, infra, newFaultRegistry(t, infra.Queries))
	adapter := dedup.NewAdapter(infra.Pool, 0, 0)

	syncKey := "budget:sync:campaign:" + campaignID.String()
	require.NoError(t, infra.Redis.SAdd(ctx, "budget:dirty_campaigns", campaignID.String()).Err())
	require.NoError(t, infra.Redis.Set(ctx, syncKey, 300_000, 0).Err())

	repo := NewCampaignRepoWithDB(infra.Pool, infra.Queries)
	worker := NewSyncWorker(infra.Redis, repo, nil, time.Hour, 0, nil, 0)
	worker.SetDedupAdapter(adapter)
	worker.SyncAll(ctx)

	var spendAfterFirst int64
	require.NoError(t, infra.Pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`, ToUUID(campaignID)).Scan(&spendAfterFirst))
	assert.Equal(t, int64(300_000), spendAfterFirst)

	var idemCount int
	require.NoError(t, infra.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM sync_idempotency`).Scan(&idemCount))
	assert.Equal(t, 1, idemCount)

	require.NoError(t, infra.Redis.SAdd(ctx, "budget:dirty_campaigns", campaignID.String()).Err())
	worker.SyncAll(ctx)

	var spendAfterReplay int64
	require.NoError(t, infra.Pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`, ToUUID(campaignID)).Scan(&spendAfterReplay))

	require.NoError(t, infra.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM sync_idempotency`).Scan(&idemCount))

	faultproof.Log(t, "dedup_crash_recovery", map[string]string{
		"subsystem":       "sync_worker",
		"spend_unchanged": strconv.FormatBool(spendAfterFirst == spendAfterReplay),
		"sync_idem_rows":  strconv.Itoa(idemCount),
		"campaign_id":     campaignID.String(),
		"baseline_ok":     strconv.FormatBool(spendAfterFirst == spendAfterReplay && idemCount == 1),
	})
}

func TestFault_DedupResumeApply(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	campaignID := seedFaultCampaign(t, infra, newFaultRegistry(t, infra.Queries))
	adapter := dedup.NewAdapter(infra.Pool, 0, 0)
	txID := uuid.New().String()

	seq := dedupkey.InflightSeq(txID)
	scope := adapter.RegionScope(dedupkey.SyncWorkerSourceID(0, campaignID), seq, seq)
	factorU := dedupkey.FactorU(dedupkey.CanonicalSpendPayload([]dedupkey.SpendPair{{
		CampaignID:  campaignID,
		AmountMicro: 125_000,
	}}))
	claim, err := adapter.ClaimConfirm(ctx, scope, factorU)
	require.NoError(t, err)
	require.Equal(t, dedup.OutcomeConfirmed, claim.Outcome)

	resume, err := adapter.NeedsResumeApply(ctx, claim.DedupKey)
	require.NoError(t, err)
	require.True(t, resume)

	repo := NewCampaignRepoWithDB(infra.Pool, infra.Queries)
	require.NoError(t, repo.UpdateSpend(ctx, campaignID, 125_000, claim.DedupKey))

	replay, err := adapter.ClaimConfirm(ctx, scope, factorU)
	require.NoError(t, err)
	assert.Equal(t, dedup.OutcomeAlreadyConfirmed, replay.Outcome)

	resume, err = adapter.NeedsResumeApply(ctx, replay.DedupKey)
	require.NoError(t, err)
	assert.False(t, resume)

	require.NoError(t, repo.UpdateSpend(ctx, campaignID, 125_000, replay.DedupKey))

	var spend int64
	require.NoError(t, infra.Pool.QueryRow(ctx, `SELECT current_spend FROM campaigns WHERE id = $1`, ToUUID(campaignID)).Scan(&spend))
	assert.Equal(t, int64(125_000), spend)

	faultproof.Log(t, "dedup_resume_apply", map[string]string{
		"subsystem":   "sync_worker",
		"spend_micro": strconv.FormatInt(spend, 10),
		"baseline_ok": "true",
	})
}
