package platformadmin

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const ledgerExportBatchLimit = 500

func formatLedgerMicro(m int64) string {
	return money.FormatFixed2(m)
}

func listCustomerLedger(ctx context.Context, host CustomerLedgerHost, customerID uuid.UUID, limit, offset int32) ([]BalanceLedgerDTO, int64, error) {
	q := db.New(host.Pool())
	tid := domain.ToUUID(customerID)
	listParams := db.ListCustomerLedgerParams{
		CustomerID: tid,
		Limit:      limit,
		Offset:     offset,
	}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountCustomerLedger(ctx, tid) },
		func() ([]db.BalanceLedger, error) { return q.ListCustomerLedger(ctx, listParams) },
		customerLedgerToDTO,
	)
}

func customerLedgerToDTO(r db.BalanceLedger) BalanceLedgerDTO {
	var campID string
	if r.CampaignID.Valid {
		campID = uuid.UUID(r.CampaignID.Bytes).String()
	}
	return BalanceLedgerDTO{
		ID:              r.ID,
		CustomerID:      uuid.UUID(r.CustomerID.Bytes).String(),
		CampaignID:      campID,
		Amount:          formatLedgerMicro(r.Amount),
		Type:            string(r.Type),
		IdempotencyHash: r.IdempotencyHash.String,
		CreatedAt:       r.CreatedAt.Time.Format(time.RFC3339),
	}
}

func getCustomerBalance(ctx context.Context, host CustomerLedgerHost, customerID uuid.UUID) (CustomerBalanceDTO, error) {
	q := db.New(host.Pool())
	cust, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID))
	if err != nil {
		return CustomerBalanceDTO{}, host.MapCustomerNotFound(err)
	}
	return CustomerBalanceDTO{
		CustomerID: customerID.String(),
		Balance:    formatLedgerMicro(cust.Balance),
		Currency:   cust.Currency,
		Ledger:     []BalanceLedgerDTO{},
	}, nil
}

func exportCustomerLedgerCSV(ctx context.Context, host CustomerLedgerHost, customerID uuid.UUID, cursor int64, w io.Writer) (LedgerExportResult, error) {
	q := db.New(host.Pool())
	if _, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID)); err != nil {
		return LedgerExportResult{}, err
	}

	limited := billingadmin.NewExportLimitedWriter(w, host.ExportChunkMaxBytes())
	cw := csv.NewWriter(limited)
	if err := cw.Write([]string{"id", "customer_id", "campaign_id", "amount", "type", "idempotency_hash", "created_at"}); err != nil {
		return LedgerExportResult{}, err
	}

	var (
		nextCursor = cursor
		truncated  bool
		lastID     int64
	)

	for {
		rows, err := q.ListCustomerLedgerExport(ctx, db.ListCustomerLedgerExportParams{
			CustomerID: domain.ToUUID(customerID),
			CursorID:   nextCursor,
			BatchLimit: ledgerExportBatchLimit,
		})
		if err != nil {
			return LedgerExportResult{}, err
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			if limited.Remaining() <= 0 {
				truncated = true
				goto done
			}
			campID := ""
			if row.CampaignID.Valid {
				campID = uuid.UUID(row.CampaignID.Bytes).String()
			}
			record := []string{
				strconv.FormatInt(row.ID, 10),
				uuid.UUID(row.CustomerID.Bytes).String(),
				campID,
				formatLedgerMicro(row.Amount),
				string(row.Type),
				row.IdempotencyHash.String,
				row.CreatedAt.Time.UTC().Format(time.RFC3339),
			}
			if err := cw.Write(record); err != nil {
				if limited.Overflow() {
					truncated = true
					goto done
				}
				return LedgerExportResult{}, err
			}
			lastID = row.ID
		}

		if len(rows) < ledgerExportBatchLimit {
			break
		}
		nextCursor = lastID
		if limited.Remaining() <= 0 {
			truncated = true
			break
		}
	}

done:
	cw.Flush()
	if err := cw.Error(); err != nil {
		if limited.Overflow() {
			truncated = true
		} else {
			return LedgerExportResult{}, err
		}
	}

	result := LedgerExportResult{
		Truncated: truncated,
		Bytes:     limited.BytesWritten(),
	}
	if truncated && lastID > 0 {
		result.NextCursor = lastID
	}
	return result, nil
}

func listDisputes(ctx context.Context, host DisputesHost, customerFilter string, limit, offset int32) (DisputeListResult, error) {
	resp, err := host.ListPaymentDisputes(ctx, customerFilter, limit, offset)
	if err != nil {
		return DisputeListResult{}, err
	}
	q := db.New(host.Pool())
	intentIDs := make([]pgtype.UUID, 0, len(resp.Disputes))
	for _, d := range resp.Disputes {
		intentID, parseErr := uuid.Parse(d.IntentID)
		if parseErr == nil {
			intentIDs = append(intentIDs, domain.ToUUID(intentID))
		}
	}
	chargebacksByIntent := make(map[uuid.UUID][]int64)
	if len(intentIDs) > 0 {
		ledgerRows, lerr := q.ListLedgerChargebackEntryIDsByIntents(ctx, intentIDs)
		if lerr == nil {
			for _, row := range ledgerRows {
				intentID := uuid.UUID(row.PaymentIntentID.Bytes)
				chargebacksByIntent[intentID] = append(chargebacksByIntent[intentID], row.ID)
			}
		}
	}

	rows := make([]DisputeRowDTO, 0, len(resp.Disputes))
	for _, d := range resp.Disputes {
		item := DisputeRowDTO{
			IntentID:          d.IntentID,
			CustomerID:        d.CustomerID,
			AmountMicro:       d.AmountMicro,
			Currency:          d.Currency,
			ProviderDisputeID: d.ProviderDisputeID,
		}
		if !d.UpdatedAt.IsZero() {
			item.UpdatedAt = d.UpdatedAt.UTC().Format(time.RFC3339)
		}
		intentID, parseErr := uuid.Parse(d.IntentID)
		if parseErr == nil {
			if ledgerIDs, ok := chargebacksByIntent[intentID]; ok && len(ledgerIDs) > 0 {
				item.ChargebackLedgerEntryIDs = ledgerIDs
			} else {
				item.ChargebackLedgerEntryIDs = []int64{}
			}
		}
		rows = append(rows, item)
	}
	return DisputeListResult{Disputes: rows, Total: resp.Total}, nil
}
