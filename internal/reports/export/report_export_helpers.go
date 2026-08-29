package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
)

type clickhouseReportRowsFunc func(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignIDs []uuid.UUID, from, to time.Time, limit, offset int) ([]map[string]any, int64, error)

func writeCSVMapRow(w *csv.Writer, row map[string]any, cols ...string) error {
	rec := make([]string, len(cols))
	for i, col := range cols {
		rec[i] = csvMapField(row, col)
	}
	return w.Write(rec)
}

func csvMapField(row map[string]any, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case int:
		return fmt.Sprintf("%d", x)
	case int32:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case uint:
		return fmt.Sprintf("%d", x)
	case uint32:
		return fmt.Sprintf("%d", x)
	case uint64:
		return fmt.Sprintf("%d", x)
	case float32:
		return fmt.Sprintf("%g", x)
	case float64:
		return fmt.Sprintf("%g", x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func exportCHMapReport(
	ctx context.Context,
	w *csv.Writer,
	clickhouseQuery *database.ClickHouseQuery,
	campaignIDs []uuid.UUID,
	from, to time.Time,
	queryFn clickhouseReportRowsFunc,
	headers []string,
	cols ...string,
) error {
	if err := w.Write(headers); err != nil {
		return err
	}
	return paginateCHExport(reportExportPageSize,
		func(offset, limit int) ([]map[string]any, int64, error) {
			return queryFn(ctx, clickhouseQuery, campaignIDs, from, to, limit, offset)
		},
		func(row map[string]any) error {
			return writeCSVMapRow(w, row, cols...)
		},
	)
}
