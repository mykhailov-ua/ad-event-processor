package ingestion

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type outageCampaignRepo struct {
	*CampaignRepo
	fail bool
}

func (r *outageCampaignRepo) UpdateSpend(ctx context.Context, id uuid.UUID, amount int64, txID string) error {
	if r.fail {
		return errors.New("simulated pg outage")
	}
	return r.CampaignRepo.UpdateSpend(ctx, id, amount, txID)
}

func (r *outageCampaignRepo) UpdateSpendBatch(ctx context.Context, items []SpendFlushItem) ([]SpendFlushOutcome, error) {
	if r.fail {
		return nil, errors.New("simulated pg outage")
	}
	return r.CampaignRepo.UpdateSpendBatch(ctx, items)
}

func (r *outageCampaignRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.CampaignStatus) error {
	return r.CampaignRepo.UpdateStatus(ctx, id, status)
}

func (r *outageCampaignRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	return r.CampaignRepo.GetByID(ctx, id)
}

func (r *outageCampaignRepo) ListActive(ctx context.Context) ([]*domain.Campaign, error) {
	return r.CampaignRepo.ListActive(ctx)
}

func TestFault_LedgerBatch_PGOutage_RollupRetained(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	ctx := context.Background()
	campaignID := seedFaultCampaign(t, infra, newFaultRegistry(t, infra.Queries))

	syncKey := "budget:sync:campaign:" + campaignID.String()
	require.NoError(t, infra.Redis.SAdd(ctx, "budget:dirty_campaigns", campaignID.String()).Err())
	require.NoError(t, infra.Redis.Set(ctx, syncKey, 250_000, 0).Err())

	baseRepo := NewCampaignRepoWithDB(infra.Pool, infra.Queries)
	repo := &outageCampaignRepo{CampaignRepo: baseRepo, fail: true}
	worker := NewSyncWorker(infra.Redis, repo, nil, time.Hour, 0, nil, 0)

	worker.SyncAll(ctx)

	assert.Equal(t, 1, worker.PendingCampaignRollupCount(), "rollup must be retained after PG outage")

	inflight, err := infra.Redis.Get(ctx, "budget:inflight:campaign:"+campaignID.String()).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(250_000), inflight, "inflight must remain until successful flush")

	faultproof.Log(t, "ledger_batch_pg_outage", map[string]string{
		"subsystem":       "ads_processor",
		"rollup_retained": strconv.FormatBool(worker.PendingCampaignRollupCount() > 0),
		"inflight_micro":  strconv.FormatInt(inflight, 10),
		"baseline_ok":     "true",
	})
}
