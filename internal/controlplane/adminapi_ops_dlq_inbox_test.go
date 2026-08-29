package controlplane

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/opsadmin"

	"github.com/stretchr/testify/require"
)

type stubOpsReaderInbox struct {
	inbox opsadmin.DLQInboxListResult
	err   error
}

func (s stubOpsReaderInbox) GetIncidentSnapshot(ctx context.Context) (opsadmin.IncidentSnapshotDTO, error) {
	return opsadmin.IncidentSnapshotDTO{}, nil
}

func (s stubOpsReaderInbox) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (opsadmin.OutboxListResult, error) {
	return opsadmin.OutboxListResult{}, nil
}

func (s stubOpsReaderInbox) ListDLQEntries(ctx context.Context, cursor string, limit int) (opsadmin.FanOutResult[opsadmin.DLQEntryDTO], error) {
	return opsadmin.FanOutResult[opsadmin.DLQEntryDTO]{}, nil
}

func (s stubOpsReaderInbox) ListDLQInbox(ctx context.Context, source, cursor string, limit int) (opsadmin.DLQInboxListResult, error) {
	return s.inbox, s.err
}

func (s stubOpsReaderInbox) RetryDLQInbox(ctx context.Context, source, id, idempotencyKey string) error {
	return nil
}

func (s stubOpsReaderInbox) EnqueueDLQRetry(ctx context.Context, payload opsadmin.DLQRetryPayload, idempotencyKey string) error {
	return nil
}

func (s stubOpsReaderInbox) ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (opsadmin.ConsentProofListResult, error) {
	return opsadmin.ConsentProofListResult{}, nil
}

func (s stubOpsReaderInbox) ListDomainRotation(ctx context.Context) (opsadmin.DomainRotationListResult, error) {
	return opsadmin.DomainRotationListResult{}, nil
}

func (s stubOpsReaderInbox) GetShardHealthFanOut(ctx context.Context) (opsadmin.ShardHealthAPIResponse, error) {
	return opsadmin.ShardHealthAPIResponse{}, nil
}

func (s stubOpsReaderInbox) ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (opsadmin.AuditExportResult, error) {
	return opsadmin.AuditExportResult{}, nil
}

func (s stubOpsReaderInbox) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	return "", nil
}

func (s stubOpsReaderInbox) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]opsadmin.ReconRunDTO, int64, error) {
	return nil, 0, nil
}

func (s stubOpsReaderInbox) GetDashboardSummary(ctx context.Context) (opsadmin.DashboardSummaryDTO, error) {
	return opsadmin.DashboardSummaryDTO{}, nil
}

func (s stubOpsReaderInbox) GetStackHealthSnapshot(ctx context.Context) (opsadmin.StackHealthSnapshot, error) {
	return opsadmin.StackHealthSnapshot{Status: "ok", LicenseState: "ACTIVE"}, nil
}

func (s stubOpsReaderInbox) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (opsadmin.DashboardMetricsDTO, error) {
	return opsadmin.DashboardMetricsDTO{}, nil
}

func (s stubOpsReaderInbox) GetMLModelStatus(ctx context.Context) (opsadmin.MLModelStatusDTO, error) {
	return opsadmin.MLModelStatusDTO{}, nil
}

func (s stubOpsReaderInbox) GetMLEvalReport(ctx context.Context) (opsadmin.MLEvalReportDTO, error) {
	return opsadmin.MLEvalReportDTO{
		Status:         "empty",
		ProxyMetrics:   opsadmin.MLEvalMetricsBlockDTO{Status: "empty", LabelMethod: "proxy", LabeledRows: 0},
		AuditedMetrics: opsadmin.DefaultEmptyAuditedMetrics(),
	}, nil
}

func (s stubOpsReaderInbox) AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error {
	return nil
}

func (s stubOpsReaderInbox) ListMLManualLabels(ctx context.Context) ([]fraudadmin.MLManualLabelDTO, error) {
	return nil, nil
}

func TestOpsHTTPHandlers_listDLQInbox_ok(t *testing.T) {
	h := &opsadmin.HTTPHandlers{
		OpsReader: stubOpsReaderInbox{inbox: opsadmin.DLQInboxListResult{
			Items: []opsadmin.DLQInboxEntryDTO{{ID: "1", Source: "postback", CampaignID: "c-1"}},
		}},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/dlq/inbox?source=postback", http.NoBody)
	rec := httptest.NewRecorder()
	h.ListDLQInbox(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"source":"postback"`)
}

type recordingOpsReaderInbox struct {
	stubOpsReaderInbox
	retrySource string
	retryID     string
}

func (s *recordingOpsReaderInbox) RetryDLQInbox(ctx context.Context, source, id, idempotencyKey string) error {
	s.retrySource = source
	s.retryID = id
	return nil
}

func TestOpsHTTPHandlers_retryDLQInbox_requiresSource(t *testing.T) {
	h := &opsadmin.HTTPHandlers{OpsReader: stubOpsReaderInbox{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/dlq/inbox/7/retry", http.NoBody)
	req.Header.Set("Idempotency-Key", "k1")
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()
	h.RetryDLQInbox(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOpsHTTPHandlers_retryDLQInbox_delegatesBySource(t *testing.T) {
	reader := &recordingOpsReaderInbox{}
	h := &opsadmin.HTTPHandlers{OpsReader: reader}
	body := `{"source":"capi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/dlq/inbox/42/retry", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k2")
	req.SetPathValue("id", "42")
	rec := httptest.NewRecorder()
	h.RetryDLQInbox(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, "capi", reader.retrySource)
	require.Equal(t, "42", reader.retryID)
}

func TestOpsHTTPHandlers_listConsentProofs_ok(t *testing.T) {
	h := &opsadmin.HTTPHandlers{
		OpsReader: stubOpsReaderInbox{},
	}

	consentStub := consentProofStub{
		result: opsadmin.ConsentProofListResult{
			Items: []opsadmin.ConsentProofDTO{{ID: 1, Source: "cmp", UserIDHash: "abc"}},
		},
	}
	h.OpsReader = consentStub
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/consent/proofs?limit=10", http.NoBody)
	rec := httptest.NewRecorder()
	h.ListConsentProofs(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"source":"cmp"`)
}

type consentProofStub struct {
	stubOpsReaderInbox
	result opsadmin.ConsentProofListResult
	err    error
}

func (s consentProofStub) ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (opsadmin.ConsentProofListResult, error) {
	return s.result, s.err
}
