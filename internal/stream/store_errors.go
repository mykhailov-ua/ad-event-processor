package stream

import (
	"context"
	"errors"
	"strings"
)

// isRetriableStoreError: true -> retain batch in PEL and backoff (CH down, spool full).
// false -> splitStoreBatch/DLQ path for schema or poison rows.
func isRetriableStoreError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, errCHSpoolMaxSegments) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connect:") ||
		strings.Contains(s, "closed pool") ||
		strings.Contains(s, "conn closed") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "clickhouse unavailable") ||
		strings.Contains(s, "clickhouse write failed") ||
		strings.Contains(s, "failed to connect")
}
