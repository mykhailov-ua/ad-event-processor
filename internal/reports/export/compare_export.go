package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
)

func parseCompareRangeFromSpec(spec reportjob.ReportJobSpec) (from, to time.Time, enabled bool, err error) {
	compareFrom := strings.TrimSpace(spec.CompareFrom)
	compareTo := strings.TrimSpace(spec.CompareTo)
	if compareFrom == "" && compareTo == "" {
		return time.Time{}, time.Time{}, false, nil
	}
	from, to, err = reportjob.ParseReportRangeFromStrings(compareFrom, compareTo)
	return from, to, true, err
}

func writeCompareMetaHeader(w *csv.Writer, compareFrom, compareTo string) error {
	if compareFrom == "" || compareTo == "" {
		return nil
	}
	if err := w.Write([]string{"# compare_from", compareFrom}); err != nil {
		return err
	}
	return w.Write([]string{"# compare_to", compareTo})
}

func buildReportMetricsPrevMap(
	ctx context.Context,
	deps reports.ReportExportDeps,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	maxRows int,
	fetch func(context.Context, reports.ReportExportDeps, []uuid.UUID, time.Time, time.Time, int, int) ([]reports.ReportMetricsCHRow, int64, error),
) (map[string]reports.ReportMetricsCHRow, error) {
	prevByKey := make(map[string]reports.ReportMetricsCHRow)
	offset := 0
	for len(prevByKey) < maxRows {
		limit := reportExportPageSize
		if remaining := maxRows - len(prevByKey); remaining < limit {
			limit = remaining
		}
		rows, total, err := fetch(ctx, deps, campaignIDs, from, to, limit, offset)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			prevByKey[reports.ReportMetricsKey(row.Dimension, row.CampaignID)] = row
		}
		offset += len(rows)
		if int64(offset) >= total || len(rows) == 0 {
			break
		}
	}
	return prevByKey, nil
}

func formatCompareDelta(current, previous int64) string {
	return fmt.Sprintf("%d", current-previous)
}
