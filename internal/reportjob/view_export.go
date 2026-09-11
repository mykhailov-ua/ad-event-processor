package reportjob

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SavedViewReportKeyBilling = "billing-export"
	SavedViewReportKeyAudit   = "audit-export"
)

type savedViewExportMeta struct {
	Kind          string          `json:"kind"`
	Format        string          `json:"format"`
	Destination   string          `json:"destination"`
	RedactPII     bool            `json:"redact_pii"`
	ImportPayload json.RawMessage `json:"import_payload,omitempty"`
	reportScheduleSpec
}

func BuildReportJobSpecFromSavedView(customerID, reportKey, exportedBy string, specJSON []byte) (ReportJobSpec, error) {
	reportKey = strings.TrimSpace(reportKey)
	if reportKey == "" {
		return ReportJobSpec{}, fmt.Errorf("report_key required")
	}
	if reportKey == SavedViewReportKeyBilling || reportKey == SavedViewReportKeyAudit {
		return ReportJobSpec{}, fmt.Errorf("saved view export supports report jobs only")
	}
	var meta savedViewExportMeta
	if len(specJSON) > 0 {
		if err := json.Unmarshal(specJSON, &meta); err != nil {
			return ReportJobSpec{}, fmt.Errorf("invalid saved view spec")
		}
	}
	kind := strings.TrimSpace(meta.Kind)
	if kind != "" && kind != "report" {
		return ReportJobSpec{}, fmt.Errorf("saved view export supports report jobs only")
	}

	rangeSpec := parseReportScheduleSpec(specJSON)
	now := time.Now().UTC()
	to := now
	from := now.Add(-7 * 24 * time.Hour)
	if rangeSpec.ToOffsetDays != 0 {
		to = now.Add(time.Duration(rangeSpec.ToOffsetDays) * 24 * time.Hour)
	}
	fromDays := rangeSpec.FromOffsetDays
	if fromDays <= 0 {
		fromDays = 7
	}
	from = now.Add(-time.Duration(fromDays) * 24 * time.Hour)
	if rangeSpec.From != "" {
		parsed, err := time.Parse(time.RFC3339, rangeSpec.From)
		if err != nil {
			return ReportJobSpec{}, fmt.Errorf("invalid spec.from")
		}
		from = parsed.UTC()
	}
	if rangeSpec.To != "" {
		parsed, err := time.Parse(time.RFC3339, rangeSpec.To)
		if err != nil {
			return ReportJobSpec{}, fmt.Errorf("invalid spec.to")
		}
		to = parsed.UTC()
	}

	format := strings.TrimSpace(meta.Format)
	if format == "" {
		format = "csv"
	}
	destination := normalizeReportScheduleDestination(meta.Destination)
	if destination == "" {
		destination = "download"
	}

	jobSpec := ReportJobSpec{
		CustomerID:    strings.TrimSpace(customerID),
		ReportKey:     reportKey,
		From:          from.Format(time.RFC3339),
		To:            to.Format(time.RFC3339),
		CompareFrom:   strings.TrimSpace(rangeSpec.CompareFrom),
		CompareTo:     strings.TrimSpace(rangeSpec.CompareTo),
		Format:        format,
		Destination:   destination,
		GoogleSheet:   rangeSpec.GoogleSheet,
		Notify:        rangeSpec.Notify,
		ExportedBy:    strings.TrimSpace(exportedBy),
		RowLimit:      rangeSpec.RowLimit,
		ImportPayload: meta.ImportPayload,
	}
	if err := validateReportJobCompareSpec(jobSpec); err != nil {
		return ReportJobSpec{}, err
	}
	if err := normalizeReportJobNotify(&jobSpec); err != nil {
		return ReportJobSpec{}, err
	}
	return jobSpec, nil
}

func (r *ReportJobRunner) ExportSavedView(
	ctx context.Context,
	customerID, reportKey, ownerUserID string,
	specJSON []byte,
	viewID string,
) (string, error) {
	if r == nil {
		return "", fmt.Errorf("report job runner unavailable")
	}
	spec, err := BuildReportJobSpecFromSavedView(customerID, reportKey, ownerUserID, specJSON)
	if err != nil {
		return "", err
	}
	if err := r.validateScheduleSheetsOAuth(ctx, spec.Destination, ownerUserID, true); err != nil {
		return "", err
	}
	idem := fmt.Sprintf("view-export:%s:%s", strings.TrimSpace(viewID), time.Now().UTC().Format("2006-01-02T15:04"))
	return r.CreateJob(ctx, spec, idem)
}
