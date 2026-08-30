package commandpalette

import (
	"log/slog"

	"github.com/google/uuid"
)

func logSearchAudit(enabled bool, customerID uuid.UUID, userID uuid.UUID, qLen int, resultCount int, kindsCount int, degraded bool) {
	if !enabled {
		return
	}
	slog.Info("command_palette_search_log",
		"customer_id", customerID.String(),
		"user_id", userID.String(),
		"q_len", qLen,
		"result_count", resultCount,
		"kinds_count", kindsCount,
		"degraded", degraded,
	)
}
