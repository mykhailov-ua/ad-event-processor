package reports

import (
	"encoding/json"
	"fmt"
)

// FormatReportCellValue renders a report map cell for admin JSON responses.
// Object and array values are JSON-encoded once at row build time so the SPA
// does not stringify them on every table cell render.
func FormatReportCellValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return fmt.Sprintf("%t", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int32:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case uint:
		return fmt.Sprintf("%d", val)
	case uint32:
		return fmt.Sprintf("%d", val)
	case uint64:
		return fmt.Sprintf("%d", val)
	case float32:
		return fmt.Sprintf("%g", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case json.Number:
		return val.String()
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	}
}

func normalizeReportCellValue(v any) any {
	if v == nil {
		return nil
	}
	switch v.(type) {
	case string, bool, int, int32, int64, uint, uint32, uint64, float32, float64, json.Number:
		return v
	default:
		return FormatReportCellValue(v)
	}
}

// NormalizeReportMapRow stringifies nested object/array cell values in place.
func NormalizeReportMapRow(row map[string]any) map[string]any {
	if len(row) == 0 {
		return row
	}
	out := make(map[string]any, len(row))
	for key, value := range row {
		out[key] = normalizeReportCellValue(value)
	}
	return out
}

// NormalizeReportMapRows applies NormalizeReportMapRow to each row.
func NormalizeReportMapRows(rows []map[string]any) []map[string]any {
	if len(rows) == 0 {
		return rows
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		out[i] = NormalizeReportMapRow(row)
	}
	return out
}

// NewReportRowsResponse builds a report rows envelope with normalized cells.
func NewReportRowsResponse(rows []map[string]any, freshness DataFreshnessDTO, nextCursor string) ReportRowsResponse {
	return ReportRowsResponse{
		Rows:       NormalizeReportMapRows(rows),
		Freshness:  freshness,
		NextCursor: nextCursor,
	}
}
