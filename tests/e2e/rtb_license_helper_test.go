package e2e_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain/db"
	ingestion "ad-event-processor/internal/ingest"
	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const e2eCampaignProtectionCols = `tls_fingerprint_block_enabled, proxy_vpn_block_enabled, cidr_block_enabled, moderator_intel_enabled, silent_reject_enabled, fraud_threshold_pass, fraud_threshold_suspect, fraud_threshold_ivt, fraud_threshold_block, freq_limit`

// insertE2EActiveCampaign inserts ACTIVE campaign with antifraud decoy flags off (avoids HTTP 200 safe-page on /track).
func insertE2EActiveCampaign(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignID, customerID uuid.UUID,
	name string,
	budgetLimitMicro int64,
) {
	t.Helper()
	_, err := pool.Exec(ctx,
		`INSERT INTO campaigns (
			id, name, status, customer_id, budget_limit, `+e2eCampaignProtectionCols+`
		) VALUES ($1, $2, 'ACTIVE', $3, $4, false, false, false, false, false, 100, 100, 100, 100, 1000000)`,
		campaignID, name, customerID, budgetLimitMicro,
	)
	require.NoError(t, err)
}

// wireE2EBudgetAndTrackIngest warms Redis budget keys and preloads unified-filter Lua for /track e2e harnesses.
func wireE2EBudgetAndTrackIngest(
	t *testing.T,
	ctx context.Context,
	queries *db.Queries,
	rdbs []redis.UniversalClient,
	sharder ingestion.Sharder,
	registry *ingestion.Registry,
	handler *ingestion.AdsPacketHandler,
	unifiedFilter *ingestion.UnifiedFilter,
) {
	t.Helper()
	budgetWarmer := ingestion.NewBudgetCacheWarmer(rdbs, sharder)
	registry.SetBudgetWarmer(budgetWarmer)
	_, err := registry.Sync(ctx)
	require.NoError(t, err)
	_, err = budgetWarmer.WarmFromRegistry(ctx, registry)
	require.NoError(t, err)
	wireTrackIngestLuaStreamForE2E(t, ctx, registry, handler, unifiedFilter)
}

// wireSettlementSeedForE2E publishes a valid feature seed when seed coupling is enabled in the test env.
func wireSettlementSeedForE2E(t *testing.T) {
	t.Helper()
	if !licensing.SeedCouplingRequired() {
		return
	}
	licensing.ResetFeatureSeedForTest()
	t.Cleanup(licensing.ResetFeatureSeedForTest)
	licensing.PublishFeatureSeed(0x1234_5678, true)
}

// wireOpenRTBLicenseForE2E enables live OpenRTB on registry for e2e harnesses.
// Matches production ConfigureTrackRtb gate: active license + OpenRTB feature bit;
// when seed coupling is on, publishes a valid feature seed and MCK OpenRTB bit.
func wireOpenRTBLicenseForE2E(t *testing.T, registry *ingestion.Registry) {
	t.Helper()
	if registry == nil {
		t.Fatal("registry is nil")
	}

	ent := licensing.Entitlements{}
	ent.Features.RtbLive = true

	if licensing.SeedCouplingRequired() {
		licensing.ResetFeatureSeedForTest()
		t.Cleanup(licensing.ResetFeatureSeedForTest)
		licensing.PublishFeatureSeed(0x1234_5678, true)
		licensing.SetMCKFeatureBitsForTest(licensing.MCKFeatureBitOpenRTB)
	}

	registry.SetFileLicenseForTest(licensing.StateActive, ent, true)
}

// wireTrackIngestLuaStreamForE2E wires /track for tests that assert on Redis stream XLen after
// unified-filter Lua XADD (no SetDeferStreamToProducer / StreamProducer defer path).
func wireTrackIngestLuaStreamForE2E(
	t *testing.T,
	ctx context.Context,
	registry *ingestion.Registry,
	handler *ingestion.AdsPacketHandler,
	unifiedFilter *ingestion.UnifiedFilter,
) {
	t.Helper()
	if registry != nil {
		registry.MarkPubSubOK()
	}
	require.NoError(t, unifiedFilter.PreloadScripts(ctx))
	handler.SetHealthProbeState(true, true)
}
