package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/controlplane/authz"
)

const (
	ExportProfileOperatorFull  = "operator_full"
	ExportProfileBuyerSummary  = "buyer_summary"
	exportProfileSupportMasked = "support_masked"
)

func resolveExportRedactionProfile(ctx context.Context) string {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return ExportProfileOperatorFull
	}
	switch snap.Mask {
	case authz.MaskMasked:
		return ExportProfileBuyerSummary
	case authz.MaskFull:
		if snap.Has("audit:read") {
			return ExportProfileOperatorFull
		}
		return ExportProfileBuyerSummary
	default:
		return exportProfileSupportMasked
	}
}

func ExportColumnsForReport(reportKey, profile string) []string {
	all := exportAllColumns(reportKey)
	if profile == ExportProfileOperatorFull || len(all) == 0 {
		return all
	}
	deny := exportDeniedColumns(reportKey, profile)
	if len(deny) == 0 {
		return all
	}
	denySet := make(map[string]struct{}, len(deny))
	for _, col := range deny {
		denySet[col] = struct{}{}
	}
	out := make([]string, 0, len(all))
	for _, col := range all {
		if _, blocked := denySet[col]; blocked {
			continue
		}
		out = append(out, col)
	}
	return out
}

func exportDeniedColumns(reportKey, profile string) []string {
	switch profile {
	case ExportProfileBuyerSummary:
		switch reportKey {
		case "postback-reconciliation":
			return []string{"click_id"}
		case "fraud-breakdown":
			return []string{"placement_id"}
		case "customer-fraud-by-type", "customer-fraud-by-dimension":
			return []string{"fraud_reason", "placement_id", "dimension_value"}
		case "filter-rejects":
			return []string{"country", "placement_id"}
		}
	case exportProfileSupportMasked:
		switch reportKey {
		case "postback-reconciliation":
			return []string{"click_id", "error_message"}
		case "fraud-breakdown":
			return []string{"placement_id", "fraud_reason"}
		}
	}
	return nil
}

func exportAllColumns(reportKey string) []string {
	switch reportKey {
	case "postback-reconciliation":
		return []string{"campaign_id", "click_id", "conversion_at", "conversion_value_micro", "ledger_day_fee_micro", "postback_status", "reconcile_status", "error_message"}
	case "fraud-breakdown":
		return []string{"campaign_id", "placement_id", "fraud_reason", "event_count", "silent_reject_count", "silent_reject_ratio"}
	case "customer-fraud-by-type":
		return []string{"campaign_id", "fraud_category", "fraud_category_label", "event_count", "silent_reject_count", "share_pct", "silent_reject_ratio"}
	case "customer-fraud-by-dimension":
		return []string{"dimension_value", "campaign_id", "impressions", "clicks", "ivt_events", "blocked_events", "ivt_rate", "top_fraud_category", "top_fraud_category_label"}
	case "filter-rejects":
		return []string{"reject_kind", "reject_count", "country", "placement_id"}
	default:
		return nil
	}
}

func projectExportRow(fullHeader []string, fullRow []string, allowedCols []string) []string {
	if len(allowedCols) == 0 {
		return fullRow
	}
	idx := make(map[string]int, len(fullHeader))
	for i, col := range fullHeader {
		idx[col] = i
	}
	out := make([]string, 0, len(allowedCols))
	for _, col := range allowedCols {
		if i, ok := idx[col]; ok && i < len(fullRow) {
			out = append(out, fullRow[i])
		}
	}
	return out
}

func customerFraudByTypeCSVRow(row reports.CustomerFraudByTypeRowDTO) []string {
	return []string{
		row.CampaignID,
		row.FraudCategory,
		row.FraudCategoryLabel,
		fmt.Sprintf("%d", row.EventCount),
		fmt.Sprintf("%d", row.SilentRejectCount),
		fmt.Sprintf("%.4f", row.SharePct),
		fmt.Sprintf("%.6f", row.SilentRejectRatio),
	}
}

func customerFraudByDimensionCSVRow(row reports.CustomerFraudByDimensionRowDTO) []string {
	return []string{
		row.DimensionValue,
		row.CampaignID,
		fmt.Sprintf("%d", row.Impressions),
		fmt.Sprintf("%d", row.Clicks),
		fmt.Sprintf("%d", row.IVTEvents),
		fmt.Sprintf("%d", row.BlockedEvents),
		fmt.Sprintf("%.6f", row.IVTRate),
		row.TopFraudCategory,
		row.TopFraudCategoryLabel,
	}
}

func writeProfiledCSVHeader(w *csv.Writer, reportKey, profile string) ([]string, error) {
	allowed := ExportColumnsForReport(reportKey, profile)
	if len(allowed) == 0 {
		return nil, fmt.Errorf("no export columns for report %q", reportKey)
	}
	return allowed, w.Write(allowed)
}

func writeCustomerFraudByTypeExport(w *csv.Writer, profile string, rows []reports.CustomerFraudByTypeRowDTO) error {
	const reportKey = "customer-fraud-by-type"
	fullHeader := exportAllColumns(reportKey)
	allowed, err := writeProfiledCSVHeader(w, reportKey, profile)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write(projectExportRow(fullHeader, customerFraudByTypeCSVRow(row), allowed)); err != nil {
			return err
		}
	}
	return nil
}

func writeCustomerFraudByDimensionExport(w *csv.Writer, profile string, rows []reports.CustomerFraudByDimensionRowDTO) error {
	const reportKey = "customer-fraud-by-dimension"
	fullHeader := exportAllColumns(reportKey)
	allowed, err := writeProfiledCSVHeader(w, reportKey, profile)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write(projectExportRow(fullHeader, customerFraudByDimensionCSVRow(row), allowed)); err != nil {
			return err
		}
	}
	return nil
}

func writeBuyerFraudExportPreamble(w *csv.Writer, freshness reports.DataFreshnessDTO) error {
	return reports.WriteBuyerFraudExportPreamble(w, freshness)
}

func writeExportMetaHeader(w *csv.Writer, exportedBy, deploymentID string) error {
	if exportedBy == "" {
		exportedBy = "system"
	}
	if deploymentID == "" {
		deploymentID = "unknown"
	}
	if err := w.Write([]string{"# exported_by", exportedBy}); err != nil {
		return err
	}
	if err := w.Write([]string{"# exported_at", time.Now().UTC().Format(time.RFC3339)}); err != nil {
		return err
	}
	return w.Write([]string{"# deployment_id", deploymentID})
}

func exportActorLabel(ctx context.Context) string {
	if reports.ExportActorLabel != nil {
		return strings.TrimSpace(reports.ExportActorLabel(ctx))
	}
	return ""
}

func exportDeploymentID() string {
	if reports.ExportDeploymentID != nil {
		return strings.TrimSpace(reports.ExportDeploymentID())
	}
	return ""
}
