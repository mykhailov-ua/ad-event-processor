package controlplane

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/opsadmin"

	"github.com/stretchr/testify/require"
)

type stubStackHealthReader struct {
	snap opsadmin.StackHealthSnapshot
	err  error
}

func (s stubStackHealthReader) GetIncidentSnapshot(ctx context.Context) (opsadmin.IncidentSnapshotDTO, error) {
	return opsadmin.IncidentSnapshotDTO{}, nil
}

func (s stubStackHealthReader) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (opsadmin.OutboxListResult, error) {
	return opsadmin.OutboxListResult{}, nil
}

func (s stubStackHealthReader) ListDLQEntries(ctx context.Context, cursor string, limit int) (opsadmin.FanOutResult[opsadmin.DLQEntryDTO], error) {
	return opsadmin.FanOutResult[opsadmin.DLQEntryDTO]{}, nil
}

func (s stubStackHealthReader) ListDLQInbox(ctx context.Context, source, cursor string, limit int) (opsadmin.DLQInboxListResult, error) {
	return opsadmin.DLQInboxListResult{}, nil
}

func (s stubStackHealthReader) RetryDLQInbox(ctx context.Context, source, id, idempotencyKey string) error {
	return nil
}

func (s stubStackHealthReader) EnqueueDLQRetry(ctx context.Context, payload opsadmin.DLQRetryPayload, idempotencyKey string) error {
	return nil
}

func (s stubStackHealthReader) ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (opsadmin.ConsentProofListResult, error) {
	return opsadmin.ConsentProofListResult{}, nil
}

func (s stubStackHealthReader) ListDomainRotation(ctx context.Context) (opsadmin.DomainRotationListResult, error) {
	return opsadmin.DomainRotationListResult{}, nil
}

func (s stubStackHealthReader) GetShardHealthFanOut(ctx context.Context) (opsadmin.ShardHealthAPIResponse, error) {
	return opsadmin.ShardHealthAPIResponse{}, nil
}

func (s stubStackHealthReader) ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (opsadmin.AuditExportResult, error) {
	return opsadmin.AuditExportResult{}, nil
}

func (s stubStackHealthReader) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	return "", nil
}

func (s stubStackHealthReader) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]opsadmin.ReconRunDTO, int64, error) {
	return nil, 0, nil
}

func (s stubStackHealthReader) GetDashboardSummary(ctx context.Context) (opsadmin.DashboardSummaryDTO, error) {
	return opsadmin.DashboardSummaryDTO{}, nil
}

func (s stubStackHealthReader) GetStackHealthSnapshot(ctx context.Context) (opsadmin.StackHealthSnapshot, error) {
	return s.snap, s.err
}

func (s stubStackHealthReader) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (opsadmin.DashboardMetricsDTO, error) {
	return opsadmin.DashboardMetricsDTO{}, nil
}

func (s stubStackHealthReader) GetMLModelStatus(ctx context.Context) (opsadmin.MLModelStatusDTO, error) {
	return opsadmin.MLModelStatusDTO{}, nil
}

func (s stubStackHealthReader) GetMLEvalReport(ctx context.Context) (opsadmin.MLEvalReportDTO, error) {
	return opsadmin.MLEvalReportDTO{}, nil
}

func (s stubStackHealthReader) AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error {
	return nil
}

func (s stubStackHealthReader) ListMLManualLabels(ctx context.Context) ([]fraudadmin.MLManualLabelDTO, error) {
	return nil, nil
}

func TestGetStackHealthSnapshot_handlerReturnsDegraded(t *testing.T) {
	ops := &opsadmin.HTTPHandlers{
		OpsReader: stubStackHealthReader{snap: opsadmin.StackHealthSnapshot{
			Status:                     "degraded",
			OutboxOldestPendingSeconds: 45,
			LicenseState:               "ACTIVE",
			RedisShardReachable:        true,
			RedisShardsReachable:       1,
			RedisShardsTotal:           1,
		}},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/health/snapshot", http.NoBody)
	rec := httptest.NewRecorder()
	ops.GetStackHealthSnapshot(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body opsadmin.StackHealthSnapshot
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, "degraded", body.Status)
	require.Equal(t, 45.0, body.OutboxOldestPendingSeconds)
	require.False(t, opsadmin.StackHealthSnapshotHasSecretMaterial(rec.Body.String()))
}
