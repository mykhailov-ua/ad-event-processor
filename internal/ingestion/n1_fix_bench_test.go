package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestN1Fix_ReconciliationCampaignSpend_QueryCount(t *testing.T) {
	const n = 50
	campaigns := make([]*domain.Campaign, n)
	spends := make(map[uuid.UUID]int64, n)
	for i := range n {
		id := uuid.New()
		campaigns[i] = &domain.Campaign{ID: id, Status: domain.CampaignStatusActive}
		spends[id] = int64((i + 1) * 1000)
	}

	pg := &MockPostgresDB{spends: spends}
	pg.Healthy.Store(true)
	ch := &MockClickHouseDB{}
	repo := &MockCampaignRepository{campaigns: campaigns}

	rw := NewReconciliationWorker(pg, ch, repo, 0.005, 5*time.Minute, 10*time.Minute)
	require.NoError(t, rw.Reconcile(context.Background()))

	before := int64(n)
	after := pg.getSpendsBatchCalls.Load()
	t.Logf("reconciliation_pg_spend queries: before=%d after=%d (campaigns=%d)", before, after, n)
	require.Equal(t, int64(1), after)
	require.Equal(t, int64(0), pg.getSpendCalls.Load())
}
