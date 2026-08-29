package fraud

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFraudEvidencePack_signAndVerify_holdout(t *testing.T) {
	secret := []byte("test-evidence-secret")
	pack := reports.FraudEvidencePackDTO{
		ClickID:    "click-1",
		CustomerID: "cust-1",
		CampaignID: "camp-1",
		Timeline: []reports.FraudEvidenceTimelineRowDTO{
			{EventType: "click", CampaignID: "camp-1", CreatedAt: "2026-08-27T12:00:00Z"},
		},
		FraudEvents: []reports.FraudEvidenceFraudRowDTO{
			{
				EventType:        "conversion",
				CampaignID:       "camp-1",
				FraudReason:      "sec_fetch_anomaly,tls_ja4_mismatch",
				FraudScore:       70,
				LayerDesyncCount: 2,
				CreatedAt:        "2026-08-27T12:01:00Z",
			},
		},
	}
	pack.Signals = aggregateFraudEvidenceSignals(pack.FraudEvents)

	signed, err := BuildSignedFraudEvidencePack(secret, pack)
	require.NoError(t, err)
	require.NotEmpty(t, signed.Signature)
	require.NotEmpty(t, signed.DigestSHA256)
	require.NoError(t, VerifyFraudEvidencePackSignature(secret, signed))

	signed.FraudEvents[0].FraudScore = 10
	assert.Error(t, VerifyFraudEvidencePackSignature(secret, signed))
}

func TestAggregateFraudEvidenceSignals_holdoutDedupesReasons(t *testing.T) {
	rows := []reports.FraudEvidenceFraudRowDTO{
		{FraudReason: "tcp_syn_os_mismatch,tls_ja4_mismatch", FraudScore: 35, LayerDesyncCount: 2},
		{FraudReason: "tls_ja4_mismatch", FraudScore: 70, LayerDesyncCount: 1, SilentRejectEvent: true},
	}
	sig := aggregateFraudEvidenceSignals(rows)
	assert.Equal(t, uint32(70), sig.MaxFraudScore)
	assert.Equal(t, uint8(2), sig.MaxLayerDesyncCount)
	assert.Equal(t, 1, sig.SilentRejectEvents)
	require.Len(t, sig.FraudReasons, 2)
	assert.Contains(t, sig.FraudReasons, "tcp_syn_os_mismatch")
	assert.Contains(t, sig.FraudReasons, "tls_ja4_mismatch")
}

func TestFraudEvidencePackQuery_selectsFraudRows_holdout(t *testing.T) {
	t.Parallel()
	require.Contains(t, fraudEvidencePackQuery, "FROM fraud_events")
	require.Contains(t, fraudEvidencePackQuery, "layer_desync_count")
	require.Contains(t, fraudEvidencePackQuery, "click_id = ?")
}

func TestFraudEvidencePackRoute_missingSecret503(t *testing.T) {
	t.Parallel()
	h := reports.NewReportsHTTPHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/fraud-evidence-pack?customer_id="+uuid.New().String()+"&click_id=clk-1", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "EVIDENCE_SIGNING_UNAVAILABLE")
}

func TestFraudEvidencePackRoute_missingClickID400(t *testing.T) {
	t.Parallel()
	h := &reports.ReportsHTTPHandlers{
		FraudEvidencePackHMACSecret: []byte("test-secret"),
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/fraud-evidence-pack?customer_id="+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "click_id required")
}
