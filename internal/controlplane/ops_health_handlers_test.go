package controlplane

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubStackHealthReader struct {
	snap StackHealthSnapshot
	err  error
}

func (s stubStackHealthReader) GetIncidentSnapshot(ctx context.Context) (IncidentSnapshotDTO, error) {
	return IncidentSnapshotDTO{}, nil
}

func (s stubStackHealthReader) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (OutboxListResult, error) {
	return OutboxListResult{}, nil
}

func (s stubStackHealthReader) ListDLQEntries(ctx context.Context, cursor string, limit int) (FanOutResult[DLQEntryDTO], error) {
	return FanOutResult[DLQEntryDTO]{}, nil
}

func (s stubStackHealthReader) ListDLQInbox(ctx context.Context, source, cursor string, limit int) (DLQInboxListResult, error) {
	return DLQInboxListResult{}, nil
}

func (s stubStackHealthReader) RetryDLQInbox(ctx context.Context, source, id, idempotencyKey string) error {
	return nil
}

func (s stubStackHealthReader) EnqueueDLQRetry(ctx context.Context, payload DLQRetryPayload, idempotencyKey string) error {
	return nil
}

func (s stubStackHealthReader) ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (ConsentProofListResult, error) {
	return ConsentProofListResult{}, nil
}

func (s stubStackHealthReader) ListDomainRotation(ctx context.Context) (DomainRotationListResult, error) {
	return DomainRotationListResult{}, nil
}

func (s stubStackHealthReader) GetShardHealthFanOut(ctx context.Context) (ShardHealthAPIResponse, error) {
	return ShardHealthAPIResponse{}, nil
}

func (s stubStackHealthReader) ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (AuditExportResult, error) {
	return AuditExportResult{}, nil
}

func (s stubStackHealthReader) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	return "", nil
}

func (s stubStackHealthReader) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error) {
	return nil, 0, nil
}

func (s stubStackHealthReader) GetDashboardSummary(ctx context.Context) (DashboardSummaryDTO, error) {
	return DashboardSummaryDTO{}, nil
}

func (s stubStackHealthReader) GetStackHealthSnapshot(ctx context.Context) (StackHealthSnapshot, error) {
	return s.snap, s.err
}

func (s stubStackHealthReader) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (DashboardMetricsDTO, error) {
	return DashboardMetricsDTO{}, nil
}

func (s stubStackHealthReader) GetMLModelStatus(ctx context.Context) (MLModelStatusDTO, error) {
	return MLModelStatusDTO{}, nil
}

func (s stubStackHealthReader) GetMLEvalReport(ctx context.Context) (MLEvalReportDTO, error) {
	return MLEvalReportDTO{}, nil
}

func (s stubStackHealthReader) AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error {
	return nil
}

func (s stubStackHealthReader) ListMLManualLabels(ctx context.Context) ([]MLManualLabelDTO, error) {
	return nil, nil
}

func TestGetStackHealthSnapshot_handlerReturnsDegraded(t *testing.T) {
	ops := &OpsHTTPHandlers{
		OpsReader: stubStackHealthReader{snap: StackHealthSnapshot{
			Status:                     "degraded",
			OutboxOldestPendingSeconds: 45,
			LicenseState:               "ACTIVE",
			RedisShardReachable:        true,
			RedisShardsReachable:       1,
			RedisShardsTotal:           1,
		}},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/health/snapshot", nil)
	rec := httptest.NewRecorder()
	ops.GetStackHealthSnapshot(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body StackHealthSnapshot
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, "degraded", body.Status)
	require.Equal(t, 45.0, body.OutboxOldestPendingSeconds)
	require.False(t, stackHealthSnapshotHasSecretMaterial(rec.Body.String()))
}
