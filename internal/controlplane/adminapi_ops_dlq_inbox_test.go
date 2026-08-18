package controlplane

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubOpsReaderInbox struct {
	inbox DLQInboxListResult
	err   error
}

func (s stubOpsReaderInbox) GetIncidentSnapshot(ctx context.Context) (IncidentSnapshotDTO, error) {
	return IncidentSnapshotDTO{}, nil
}

func (s stubOpsReaderInbox) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (OutboxListResult, error) {
	return OutboxListResult{}, nil
}

func (s stubOpsReaderInbox) ListDLQEntries(ctx context.Context, cursor string, limit int) (FanOutResult[DLQEntryDTO], error) {
	return FanOutResult[DLQEntryDTO]{}, nil
}

func (s stubOpsReaderInbox) ListDLQInbox(ctx context.Context, source, cursor string, limit int) (DLQInboxListResult, error) {
	return s.inbox, s.err
}

func (s stubOpsReaderInbox) RetryDLQInbox(ctx context.Context, source, id, idempotencyKey string) error {
	return nil
}

func (s stubOpsReaderInbox) EnqueueDLQRetry(ctx context.Context, payload DLQRetryPayload, idempotencyKey string) error {
	return nil
}

func (s stubOpsReaderInbox) ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (ConsentProofListResult, error) {
	return ConsentProofListResult{}, nil
}

func (s stubOpsReaderInbox) ListDomainRotation(ctx context.Context) (DomainRotationListResult, error) {
	return DomainRotationListResult{}, nil
}

func (s stubOpsReaderInbox) GetShardHealthFanOut(ctx context.Context) (ShardHealthAPIResponse, error) {
	return ShardHealthAPIResponse{}, nil
}

func (s stubOpsReaderInbox) ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (AuditExportResult, error) {
	return AuditExportResult{}, nil
}

func (s stubOpsReaderInbox) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	return "", nil
}

func (s stubOpsReaderInbox) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error) {
	return nil, 0, nil
}

func (s stubOpsReaderInbox) GetDashboardSummary(ctx context.Context) (DashboardSummaryDTO, error) {
	return DashboardSummaryDTO{}, nil
}

func (s stubOpsReaderInbox) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (DashboardMetricsDTO, error) {
	return DashboardMetricsDTO{}, nil
}

func (s stubOpsReaderInbox) GetMLModelStatus(ctx context.Context) (MLModelStatusDTO, error) {
	return MLModelStatusDTO{}, nil
}

func (s stubOpsReaderInbox) GetMLEvalReport(ctx context.Context) (MLEvalReportDTO, error) {
	return MLEvalReportDTO{
		Status:         "empty",
		ProxyMetrics:   MLEvalMetricsBlockDTO{Status: "empty", LabelMethod: "proxy", LabeledRows: 0},
		AuditedMetrics: defaultEmptyAuditedMetrics(),
	}, nil
}

func (s stubOpsReaderInbox) AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error {
	return nil
}

func (s stubOpsReaderInbox) ListMLManualLabels(ctx context.Context) ([]MLManualLabelDTO, error) {
	return nil, nil
}

func TestOpsHTTPHandlers_listDLQInbox_ok(t *testing.T) {
	h := &OpsHTTPHandlers{
		OpsReader: stubOpsReaderInbox{inbox: DLQInboxListResult{
			Items: []DLQInboxEntryDTO{{ID: "1", Source: "postback", CampaignID: "c-1"}},
		}},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/dlq/inbox?source=postback", http.NoBody)
	rec := httptest.NewRecorder()
	h.listDLQInbox(rec, req)
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
	h := &OpsHTTPHandlers{OpsReader: stubOpsReaderInbox{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/dlq/inbox/7/retry", http.NoBody)
	req.Header.Set("Idempotency-Key", "k1")
	req.SetPathValue("id", "7")
	rec := httptest.NewRecorder()
	h.retryDLQInbox(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOpsHTTPHandlers_retryDLQInbox_delegatesBySource(t *testing.T) {
	reader := &recordingOpsReaderInbox{}
	h := &OpsHTTPHandlers{OpsReader: reader}
	body := `{"source":"capi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/dlq/inbox/42/retry", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k2")
	req.SetPathValue("id", "42")
	rec := httptest.NewRecorder()
	h.retryDLQInbox(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, "capi", reader.retrySource)
	require.Equal(t, "42", reader.retryID)
}

func TestOpsHTTPHandlers_listConsentProofs_ok(t *testing.T) {
	h := &OpsHTTPHandlers{
		OpsReader: stubOpsReaderInbox{},
	}

	consentStub := consentProofStub{
		result: ConsentProofListResult{
			Items: []ConsentProofDTO{{ID: 1, Source: "cmp", UserIDHash: "abc"}},
		},
	}
	h.OpsReader = consentStub
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/consent/proofs?limit=10", http.NoBody)
	rec := httptest.NewRecorder()
	h.listConsentProofs(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"source":"cmp"`)
}

type consentProofStub struct {
	stubOpsReaderInbox
	result ConsentProofListResult
	err    error
}

func (s consentProofStub) ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (ConsentProofListResult, error) {
	return s.result, s.err
}
