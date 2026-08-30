package compat

import "ad-event-processor/internal/filter"

func init() {
	filter.SetWriteAuditLog(WriteAuditLog)
}
