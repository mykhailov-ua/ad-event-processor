package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEnqueueFraudThreatBatch_insertsOutboxRows(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fraud threat batch enqueue (run make test-integration)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	svc := newBareService(t, pool, nil, &config.Config{})
	campaignID := uuid.New().String()

	const n = 5
	items := make([]opsadmin.FraudThreatEnqueueItem, n)
	for i := range n {
		items[i] = opsadmin.FraudThreatEnqueueItem{
			Action:     "blacklist",
			IP:         fmt.Sprintf("203.0.113.%d", i),
			CampaignID: campaignID,
			Score:      90,
			TTLSeconds: 3600,
		}
	}

	inserted, err := svc.EnqueueFraudThreatBatch(ctx, items)
	require.NoError(t, err)
	require.Equal(t, n, inserted)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE event_type = 'ML_BLACKLIST_ADD'`).Scan(&count))
	require.Equal(t, n, count)
}

type stubFraudThreatEnqueuer struct {
	batchCalls int
}

func (s *stubFraudThreatEnqueuer) EnqueueFraudThreat(context.Context, string, string, string, float64, int32, int64) error {
	return nil
}

func (s *stubFraudThreatEnqueuer) EnqueueFraudThreatBatch(_ context.Context, items []opsadmin.FraudThreatEnqueueItem) (int, error) {
	s.batchCalls++
	return len(items), nil
}

func TestFraudThreatHTTP_batchBody(t *testing.T) {
	t.Parallel()

	stub := &stubFraudThreatEnqueuer{}
	ops := &opsadmin.HTTPHandlers{FraudThreat: stub}
	mux := http.NewServeMux()
	ops.RegisterFraudThreatRoutes(mux)

	body := `{"items":[{"action":"boost","ip":"1.2.3.4","campaign_id":"` + uuid.New().String() + `","score":40,"boost":40,"ttl_seconds":300}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/fraud-threat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, stub.batchCalls)
}

func TestFraudThreatHTTP_rejectsOversizeBody(t *testing.T) {
	t.Parallel()

	ops := &opsadmin.HTTPHandlers{FraudThreat: &stubFraudThreatEnqueuer{}}
	mux := http.NewServeMux()
	ops.RegisterFraudThreatRoutes(mux)

	body := strings.Repeat("x", coldpath.DefaultMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/fraud-threat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEnqueueFraudThreatBatch_rejectsOverLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fraud threat batch limit (run make test-integration)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	svc := newBareService(t, pool, nil, &config.Config{})
	campaignID := uuid.New().String()
	items := make([]opsadmin.FraudThreatEnqueueItem, fraudadmin.ThreatBatchMax+1)
	for i := range items {
		items[i] = opsadmin.FraudThreatEnqueueItem{
			Action:     "boost",
			IP:         "1.2.3.4",
			CampaignID: campaignID,
		}
	}

	_, err := svc.EnqueueFraudThreatBatch(context.Background(), items)
	require.Error(t, err)
}

func TestFraudThreatHTTP_batchJSONRoundTrip(t *testing.T) {
	t.Parallel()

	campaignID := uuid.New().String()
	raw, err := json.Marshal(map[string]any{
		"items": []map[string]any{{
			"action":      "silent_reject",
			"ip":          "198.51.100.1",
			"campaign_id": campaignID,
			"score":       88,
			"ttl_seconds": 120,
		}},
	})
	require.NoError(t, err)

	stub := &stubFraudThreatEnqueuer{}
	ops := &opsadmin.HTTPHandlers{FraudThreat: stub}
	mux := http.NewServeMux()
	ops.RegisterFraudThreatRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/fraud-threat", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, stub.batchCalls)
}
