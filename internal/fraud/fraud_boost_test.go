package fraud

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFraudManagement struct {
	batchCalls int
	enqueued   []struct {
		action     string
		ip         string
		campaignID string
		score      float64
		boost      int32
		ttlSeconds int64
	}
}

func (m *mockFraudManagement) BlockIP(ctx context.Context, ip string) error {
	return nil
}

func (m *mockFraudManagement) EnqueueFraudThreat(ctx context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	_, err := m.EnqueueFraudThreatBatch(ctx, []FraudThreatEnqueueItem{{
		Action:     action,
		IP:         ip,
		CampaignID: campaignID,
		Score:      score,
		Boost:      boost,
		TTLSeconds: ttlSeconds,
	}})
	return err
}

func (m *mockFraudManagement) EnqueueFraudThreatBatch(_ context.Context, items []FraudThreatEnqueueItem) (int, error) {
	m.batchCalls++
	for _, item := range items {
		m.enqueued = append(m.enqueued, struct {
			action     string
			ip         string
			campaignID string
			score      float64
			boost      int32
			ttlSeconds int64
		}{item.Action, item.IP, item.CampaignID, item.Score, item.Boost, item.TTLSeconds})
	}
	return len(items), nil
}

func TestDetector_FraudBoostEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	campaignID := uuid.New().String()
	stub := stubFinder{
		ips: []SuspiciousIP{
			{
				IP:         "1.2.3.4",
				Reason:     "lgbm-v1",
				Score:      45.0,
				CampaignID: campaignID,
				Action:     "boost",
				Boost:      45,
				TTLSeconds: 300,
			},
			{
				IP:     "5.6.7.8",
				Reason: "high_click_to_imp_ratio",
				Score:  90.0,
			},
		},
	}

	mgmt := &mockFraudManagement{}
	idem := NewIdempotencyStore(pool)
	detector := NewDetector(stub, idem, mgmt, pool, DetectorConfig{})

	ctx := context.Background()
	res, err := detector.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 2, res.Candidates)
	assert.Equal(t, 2, res.Enqueued)

	require.Len(t, mgmt.enqueued, 1)
	assert.Equal(t, 1, mgmt.batchCalls)
	assert.Equal(t, "boost", mgmt.enqueued[0].action)
	assert.Equal(t, "1.2.3.4", mgmt.enqueued[0].ip)
	assert.Equal(t, campaignID, mgmt.enqueued[0].campaignID)
	assert.Equal(t, 45.0, mgmt.enqueued[0].score)
	assert.Equal(t, int32(45), mgmt.enqueued[0].boost)
	assert.Equal(t, int64(300), mgmt.enqueued[0].ttlSeconds)

	res2, err := detector.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, res2.Candidates)
	assert.Equal(t, 0, res2.Enqueued)
	assert.Equal(t, 2, res2.Skipped)
}
