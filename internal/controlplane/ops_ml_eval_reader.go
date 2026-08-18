package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

const opsMLEvalReportID = "shadow_eval"

func defaultEmptyAuditedMetrics() MLEvalMetricsBlockDTO {
	return MLEvalMetricsBlockDTO{
		Status:      "empty",
		LabelMethod: "manual",
		LabeledRows: 0,
		Confidence:  "low",
	}
}

func normalizeMLEvalReport(out MLEvalReportDTO) MLEvalReportDTO {
	if out.ProxyMetrics.LabelMethod == "" && out.ProxyMetrics.LabeledRows == 0 && out.ProxyMetrics.Status == "" {
		out.ProxyMetrics = MLEvalMetricsBlockDTO{
			Status:      "empty",
			LabelMethod: "proxy",
			LabeledRows: 0,
		}
	}
	if out.AuditedMetrics.Status == "" && out.AuditedMetrics.LabelMethod == "" {
		out.AuditedMetrics = defaultEmptyAuditedMetrics()
	}
	if out.AuditedMetrics.Confidence == "" {
		if out.AuditedMetrics.LabeledRows < 30 {
			out.AuditedMetrics.Confidence = "low"
		}
	}
	return out
}

func parseMLEvalReportJSON(data []byte) (MLEvalReportDTO, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return MLEvalReportDTO{}, err
	}

	var out MLEvalReportDTO
	if v, ok := raw["status"]; ok {
		_ = json.Unmarshal(v, &out.Status)
	}
	if v, ok := raw["generated_at"]; ok {
		_ = json.Unmarshal(v, &out.GeneratedAt)
	}
	if v, ok := raw["hours"]; ok {
		_ = json.Unmarshal(v, &out.Hours)
	}
	if v, ok := raw["threshold"]; ok {
		_ = json.Unmarshal(v, &out.Threshold)
	}
	if v, ok := raw["drift"]; ok {
		out.Drift = v
	}
	if v, ok := raw["drift_detected"]; ok {
		_ = json.Unmarshal(v, &out.DriftDetected)
	}
	if v, ok := raw["proxy_metrics"]; ok {
		_ = json.Unmarshal(v, &out.ProxyMetrics)
	} else {
		_ = json.Unmarshal(data, &out.ProxyMetrics)
		if out.ProxyMetrics.Status != "" {
			out.ProxyMetrics.LabelMethod = "proxy"
		}
	}
	if v, ok := raw["audited_metrics"]; ok {
		_ = json.Unmarshal(v, &out.AuditedMetrics)
	}
	return normalizeMLEvalReport(out), nil
}

func (r *opsReader) GetMLEvalReport(ctx context.Context) (MLEvalReportDTO, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return MLEvalReportDTO{}, fmt.Errorf("postgres pool not configured")
	}

	var reportJSON []byte
	err := r.svc.GetPool().QueryRow(ctx, `
		SELECT report_json
		FROM ml_eval_reports
		WHERE id = $1`,
		opsMLEvalReportID,
	).Scan(&reportJSON)
	if err == nil && len(reportJSON) > 0 && string(reportJSON) != "{}" && string(reportJSON) != "null" {
		parsed, parseErr := parseMLEvalReportJSON(reportJSON)
		if parseErr == nil {
			return parsed, nil
		}
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return MLEvalReportDTO{}, fmt.Errorf("query ml_eval_reports: %w", err)
	}

	path := os.Getenv("FRAUD_EVAL_REPORT_PATH")
	if path == "" {
		path = "var/fraudscore/shadow_eval_report.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return normalizeMLEvalReport(MLEvalReportDTO{
			Status:         "eval_unavailable",
			ProxyMetrics:   MLEvalMetricsBlockDTO{Status: "empty", LabelMethod: "proxy", LabeledRows: 0},
			AuditedMetrics: defaultEmptyAuditedMetrics(),
		}), nil
	}
	parsed, err := parseMLEvalReportJSON(data)
	if err != nil {
		return MLEvalReportDTO{}, fmt.Errorf("parse eval report: %w", err)
	}
	return parsed, nil
}
