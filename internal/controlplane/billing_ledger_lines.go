package controlplane

import (
	"fmt"
	"time"

	billingdb "github.com/bidshard/ad-event-processor/internal/ledger/db"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
)

func ledgerLineFromRow(row billingdb.ListCustomerLedgerInWindowRow) LedgerLineDTO {
	createdAt := ""
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return LedgerLineDTO{
		ID:          row.ID,
		AmountMicro: row.Amount,
		LedgerType:  string(row.Type),
		CreatedAt:   createdAt,
	}
}

func mapLedgerLines(rows []billingdb.ListCustomerLedgerInWindowRow, limit int32) ([]LedgerLineDTO, string) {
	out := coldpath.MapSlice(rows, ledgerLineFromRow)
	nextCursor := ""
	if int32(len(out)) == limit && len(out) > 0 {
		nextCursor = fmt.Sprintf("%d", out[len(out)-1].ID)
	}
	return out, nextCursor
}
