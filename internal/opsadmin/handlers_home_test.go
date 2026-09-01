package opsadmin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/doctor"
	"ad-event-processor/internal/fraudadmin"

	"github.com/stretchr/testify/require"
)

type stubOpsHomeReader struct {
	snap StackHealthSnapshot
}

func (s stubOpsHomeReader) GetIncidentSnapshot(ctx context.Context) (IncidentSnapshotDTO, error) {
	return IncidentSnapshotDTO{}, nil
}

func (s stubOpsHomeReader) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (OutboxListResult, error) {
	return OutboxListResult{}, nil
}

func (s stubOpsHomeReader) ListDLQEntries(ctx context.Context, cursor string, limit int) (FanOutResult[DLQEntryDTO], error) {
	return FanOutResult[DLQEntryDTO]{}, nil
}

func (s stubOpsHomeReader) ListDLQInbox(ctx context.Context, source, cursor string, limit int) (DLQInboxListResult, error) {
	return DLQInboxListResult{}, nil
}

func (s stubOpsHomeReader) RetryDLQInbox(ctx context.Context, source, id, idempotencyKey string) error {
	return nil
}

func (s stubOpsHomeReader) EnqueueDLQRetry(ctx context.Context, payload DLQRetryPayload, idempotencyKey string) error {
	return nil
}

func (s stubOpsHomeReader) ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (ConsentProofListResult, error) {
	return ConsentProofListResult{}, nil
}

func (s stubOpsHomeReader) ListDomainRotation(ctx context.Context) (DomainRotationListResult, error) {
	return DomainRotationListResult{}, nil
}

func (s stubOpsHomeReader) GetShardHealthFanOut(ctx context.Context) (ShardHealthAPIResponse, error) {
	return ShardHealthAPIResponse{}, nil
}

func (s stubOpsHomeReader) ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (AuditExportResult, error) {
	return AuditExportResult{}, nil
}

func (s stubOpsHomeReader) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	return "", nil
}

func (s stubOpsHomeReader) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error) {
	return nil, 0, nil
}

func (s stubOpsHomeReader) GetDashboardSummary(ctx context.Context) (DashboardSummaryDTO, error) {
	return DashboardSummaryDTO{}, nil
}

func (s stubOpsHomeReader) GetStackHealthSnapshot(ctx context.Context) (StackHealthSnapshot, error) {
	return s.snap, nil
}

func (s stubOpsHomeReader) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (DashboardMetricsDTO, error) {
	return DashboardMetricsDTO{}, nil
}

func (s stubOpsHomeReader) GetMLModelStatus(ctx context.Context) (MLModelStatusDTO, error) {
	return MLModelStatusDTO{}, nil
}

func (s stubOpsHomeReader) GetMLEvalReport(ctx context.Context) (MLEvalReportDTO, error) {
	return MLEvalReportDTO{}, nil
}

func (s stubOpsHomeReader) AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error {
	return nil
}

func (s stubOpsHomeReader) ListMLManualLabels(ctx context.Context) ([]fraudadmin.MLManualLabelDTO, error) {
	return nil, nil
}

func TestGetOpsHome_returnsCompositeSnapshot(t *testing.T) {
	ops := &HTTPHandlers{
		OpsReader: stubOpsHomeReader{snap: StackHealthSnapshot{
			Status:               "ok",
			LicenseState:         "ACTIVE",
			RedisShardReachable:  true,
			RedisShardsReachable: 1,
			RedisShardsTotal:     1,
		}},
		DoctorSnapshot: func(ctx context.Context) (doctor.DoctorResponseDTO, error) {
			return doctor.DoctorResponseDTO{
				Overall:        "ok",
				TrackingDomain: "track.example.test",
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/home", http.NoBody)
	rec := httptest.NewRecorder()
	ops.GetOpsHome(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Doctor struct {
			Overall string `json:"overall"`
		} `json:"doctor"`
		StackHealth struct {
			Status string `json:"status"`
		} `json:"stackHealth"`
		DashboardSummary struct {
			GeneratedAt string `json:"generated_at"`
		} `json:"dashboardSummary"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, "ok", body.Doctor.Overall)
	require.Equal(t, "ok", body.StackHealth.Status)
}

func TestGetOpsHome_holdoutRequiresDoctorSnapshot(t *testing.T) {
	ops := &HTTPHandlers{
		OpsReader: stubOpsHomeReader{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/home", http.NoBody)
	rec := httptest.NewRecorder()
	ops.GetOpsHome(rec, req)
	require.NotEqual(t, http.StatusOK, rec.Code)
}
