package controlplane

import (
	"ad-event-processor/internal/opsadmin"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMLEvalReportJSON_mixedBlocks(t *testing.T) {
	raw := []byte(`{
		"status": "ok",
		"generated_at": "2026-01-01T00:00:00Z",
		"hours": 168,
		"threshold": 0.6,
		"proxy_metrics": {
			"status": "ok",
			"label_method": "proxy",
			"labeled_rows": 120,
			"precision": 0.8,
			"recall": 0.7
		},
		"audited_metrics": {
			"status": "empty",
			"label_method": "manual",
			"labeled_rows": 0,
			"confidence": "low"
		}
	}`)
	out, err := opsadmin.ParseMLEvalReportJSON(raw)
	require.NoError(t, err)
	require.Equal(t, "ok", out.Status)
	require.Equal(t, int64(120), out.ProxyMetrics.LabeledRows)
	require.Equal(t, "proxy", out.ProxyMetrics.LabelMethod)
	require.Equal(t, int64(0), out.AuditedMetrics.LabeledRows)
	require.Equal(t, "low", out.AuditedMetrics.Confidence)
}

func TestParseMLEvalReportJSON_legacyTopLevelFillsProxy(t *testing.T) {
	raw := []byte(`{
		"status": "ok",
		"labeled_rows": 50,
		"label_method": "proxy",
		"precision": 0.5,
		"recall": 0.4
	}`)
	out, err := opsadmin.ParseMLEvalReportJSON(raw)
	require.NoError(t, err)
	require.Equal(t, int64(0), out.AuditedMetrics.LabeledRows)
	require.Equal(t, "manual", out.AuditedMetrics.LabelMethod)
	require.Equal(t, "low", out.AuditedMetrics.Confidence)
}

func TestNormalizeMLEvalReport_alwaysIncludesAuditedRows(t *testing.T) {
	out := opsadmin.NormalizeMLEvalReport(opsadmin.MLEvalReportDTO{})
	require.Equal(t, int64(0), out.AuditedMetrics.LabeledRows)
	require.Equal(t, "low", out.AuditedMetrics.Confidence)
}

func TestParseMLEvalReportJSON_roundTrip(t *testing.T) {
	in := opsadmin.MLEvalReportDTO{
		Status:      "ok",
		GeneratedAt: "2026-03-01T10:00:00Z",
		Hours:       24,
		Threshold:   0.6,
		ProxyMetrics: opsadmin.MLEvalMetricsBlockDTO{
			Status:      "ok",
			LabelMethod: "proxy",
			LabeledRows: 10,
			Precision:   0.9,
		},
		AuditedMetrics: opsadmin.DefaultEmptyAuditedMetrics(),
	}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	out, err := opsadmin.ParseMLEvalReportJSON(data)
	require.NoError(t, err)
	require.Equal(t, in.ProxyMetrics.LabeledRows, out.ProxyMetrics.LabeledRows)
	require.Equal(t, in.AuditedMetrics.LabeledRows, out.AuditedMetrics.LabeledRows)
}
