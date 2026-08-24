package fraud

import (
	"context"
	"sync/atomic"
	"testing"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type batchOnlyManagement struct {
	calls atomic.Int32
	lastN int
}

func (m *batchOnlyManagement) BlockIP(context.Context, string) error {
	return nil
}

func (m *batchOnlyManagement) EnqueueFraudThreat(context.Context, string, string, string, float64, int32, int64) error {
	return nil
}

func (m *batchOnlyManagement) EnqueueFraudThreatBatch(_ context.Context, items []FraudThreatEnqueueItem) (int, error) {
	m.calls.Add(1)
	m.lastN = len(items)
	return len(items), nil
}

func TestDetector_oneHTTPBatchPerScan(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: detector batch enqueue (run make test-integration)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	campaignID := uuid.New().String()
	stub := stubFinder{
		ips: []SuspiciousIP{
			{IP: "203.0.113.1", Reason: "lgbm-v1", Score: 40, CampaignID: campaignID, Action: "boost", Boost: 40, TTLSeconds: 300},
			{IP: "203.0.113.2", Reason: "lgbm-v1", Score: 50, CampaignID: campaignID, Action: "boost", Boost: 50, TTLSeconds: 300},
			{IP: "203.0.113.3", Reason: "lgbm-v1", Score: 60, CampaignID: campaignID, Action: "silent_reject", TTLSeconds: 120},
		},
	}

	mgmt := &batchOnlyManagement{}
	detector := NewDetector(stub, NewIdempotencyStore(pool), mgmt, pool, DetectorConfig{})

	result, err := detector.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, result.Candidates)
	assert.Equal(t, 3, result.Enqueued)
	assert.Equal(t, int32(1), mgmt.calls.Load())
	assert.Equal(t, 3, mgmt.lastN)
}
