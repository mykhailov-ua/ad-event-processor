package export

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"ad-event-processor/internal/reports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCustomerFraudByTypeExport_buyerProfileOmitsRawReason_holdout(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	rows := []reports.CustomerFraudByTypeRowDTO{{
		CampaignID:         "camp-1",
		FraudCategory:      "invalid_device_signals",
		FraudCategoryLabel: "Invalid device signals",
		EventCount:         42,
		SilentRejectCount:  3,
		SharePct:           0.75,
		SilentRejectRatio:  0.0714,
	}}
	require.NoError(t, writeCustomerFraudByTypeExport(w, ExportProfileBuyerSummary, rows))
	w.Flush()
	content := buf.String()
	assert.Contains(t, content, "fraud_category")
	assert.Contains(t, content, "Invalid device signals")
	assert.NotContains(t, content, "fraud_reason")
	assert.NotContains(t, content, "tls_ja4_mismatch")
}

func TestWriteCustomerFraudByDimensionExport_buyerProfileOmitsPlacementID_holdout(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	rows := []reports.CustomerFraudByDimensionRowDTO{{
		DimensionValue:        "secret-placement-99",
		CampaignID:            "camp-1",
		Impressions:           1000,
		Clicks:                100,
		IVTEvents:             25,
		BlockedEvents:         10,
		IVTRate:               0.25,
		TopFraudCategory:      "invalid_device_signals",
		TopFraudCategoryLabel: "Invalid device signals",
	}}
	require.NoError(t, writeCustomerFraudByDimensionExport(w, ExportProfileBuyerSummary, rows))
	w.Flush()
	content := buf.String()
	assert.NotContains(t, content, "secret-placement-99")
	assert.NotContains(t, content, "dimension_value")
	assert.Contains(t, content, "camp-1")
	assert.Contains(t, content, "top_fraud_category")
}

func TestWriteCustomerFraudByDimensionExport_operatorFullIncludesPlacementID(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	rows := []reports.CustomerFraudByDimensionRowDTO{{
		DimensionValue: "placement-42",
		CampaignID:     "camp-1",
		Impressions:    500,
		Clicks:         50,
		IVTEvents:      5,
		BlockedEvents:  2,
		IVTRate:        0.1,
	}}
	require.NoError(t, writeCustomerFraudByDimensionExport(w, ExportProfileOperatorFull, rows))
	w.Flush()
	records, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "dimension_value", records[0][0])
	assert.Equal(t, "placement-42", records[1][0])
}

func TestLiveReportExportKeys_includesCustomerFraudReports(t *testing.T) {
	t.Parallel()
	keys := reports.LiveReportExportKeys()
	assert.Contains(t, keys, "customer-fraud-by-type")
	assert.Contains(t, keys, "customer-fraud-by-dimension")
}
