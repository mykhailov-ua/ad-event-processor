package reportjob

import (
	"context"
	"errors"
	"strings"
)

const (
	exportErrTimeoutPublic = "Export timed out while waiting for data. Narrow the date range or lower the row limit, then try again."
	exportErrSourcePublic  = "The report data source failed. Try a smaller date range or contact support if this persists."
	exportErrGenericPublic = "Export failed. Try again or contact support."
)

func SanitizeExportJobError(stored string) string {
	trimmed := strings.TrimSpace(stored)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "context deadline") ||
		strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "timeout") {
		return exportErrTimeoutPublic
	}
	if strings.Contains(lower, "clickhouse") ||
		strings.Contains(lower, "db::") ||
		strings.Contains(lower, "code: ") ||
		strings.Contains(lower, "memory limit") ||
		strings.Contains(lower, "too many rows") {
		return exportErrSourcePublic
	}
	if isSafeExportValidationMessage(trimmed) {
		return trimmed
	}
	if strings.Contains(lower, "invalid") ||
		strings.Contains(lower, "required") ||
		strings.Contains(lower, "must be") ||
		strings.Contains(lower, "exceeds") {
		return trimmed
	}
	return exportErrGenericPublic
}

func SanitizeExportJobErrorFromErr(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return exportErrTimeoutPublic
	}
	return SanitizeExportJobError(err.Error())
}

func isSafeExportValidationMessage(msg string) bool {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "invalid customer_id"),
		strings.Contains(lower, "report_key required"),
		strings.Contains(lower, "invalid from timestamp"),
		strings.Contains(lower, "invalid to timestamp"),
		strings.Contains(lower, "from must be before to"),
		strings.Contains(lower, "range exceeds"),
		strings.Contains(lower, "format must be"):
		return true
	default:
		return false
	}
}
