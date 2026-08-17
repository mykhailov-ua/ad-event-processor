package controlplane

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CustomerDTO = adminapi.CustomerDTO

type LedgerDTO = adminapi.BalanceLedgerDTO

type CustomerBalanceDTO = adminapi.CustomerBalanceDTO

type LedgerListResponse = adminapi.LedgerListResponse

type LedgerExportResult = adminapi.LedgerExportResult

const (
	ledgerExportMaxBytes   = 10 * 1024 * 1024
	ledgerExportBatchLimit = 500
)

func formatMicro(m int64) string {
	return money.FormatFixed2(m)
}

func (s *Service) ListCustomers(ctx context.Context, limit, offset int32) ([]CustomerDTO, int64, error) {
	q := db.New(s.GetPool())
	rows, total, err := coldpath.PaginatedQuery(
		func() (int64, error) { return q.CountCustomers(ctx) },
		func() ([]db.Customer, error) {
			return q.ListCustomers(ctx, db.ListCustomersParams{Limit: limit, Offset: offset})
		},
	)
	if err != nil {
		return nil, 0, err
	}

	var customerIDs []pgtype.UUID
	for _, r := range rows {
		customerIDs = append(customerIDs, r.ID)
	}

	stats, err := q.GetCustomerStats(ctx, customerIDs)
	if err != nil {
		return nil, 0, err
	}

	statsMap := make(map[uuid.UUID]db.GetCustomerStatsRow, len(stats))
	for _, st := range stats {
		if st.CustomerID.Valid {
			statsMap[uuid.UUID(st.CustomerID.Bytes)] = st
		}
	}

	return coldpath.MapSlice(rows, func(r db.Customer) CustomerDTO {
		uid := uuid.UUID(r.ID.Bytes)
		st := statsMap[uid]
		return CustomerDTO{
			ID:              uid.String(),
			Name:            r.Name,
			Balance:         formatMicro(r.Balance),
			Currency:        r.Currency,
			ActiveCampaigns: st.ActiveCampaigns,
			TotalSpend:      formatMicro(st.TotalSpend),
			CreatedAt:       r.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:       r.UpdatedAt.Time.Format(time.RFC3339),
		}
	}), total, nil
}

func (s *Service) GetCustomerDTO(ctx context.Context, id uuid.UUID) (CustomerDTO, error) {
	q := db.New(s.GetPool())
	r, err := q.GetCustomerByID(ctx, domain.ToUUID(id))
	if err != nil {
		return CustomerDTO{}, mapNotFound(err, ErrCustomerNotFound)
	}

	stats, err := q.GetCustomerStats(ctx, []pgtype.UUID{r.ID})
	if err != nil {
		return CustomerDTO{}, err
	}

	var st db.GetCustomerStatsRow
	if len(stats) > 0 {
		st = stats[0]
	}

	return CustomerDTO{
		ID:              uuid.UUID(r.ID.Bytes).String(),
		Name:            r.Name,
		Balance:         formatMicro(r.Balance),
		Currency:        r.Currency,
		ActiveCampaigns: st.ActiveCampaigns,
		TotalSpend:      formatMicro(st.TotalSpend),
		CreatedAt:       r.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       r.UpdatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *Service) ListCustomerLedger(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]LedgerDTO, int64, error) {
	q := db.New(s.GetPool())
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

func customerLedgerToDTO(r db.BalanceLedger) LedgerDTO {
	var campID string
	if r.CampaignID.Valid {
		campID = uuid.UUID(r.CampaignID.Bytes).String()
	}
	return LedgerDTO{
		ID:              r.ID,
		CustomerID:      uuid.UUID(r.CustomerID.Bytes).String(),
		CampaignID:      campID,
		Amount:          formatMicro(r.Amount),
		Type:            string(r.Type),
		IdempotencyHash: r.IdempotencyHash.String,
		CreatedAt:       r.CreatedAt.Time.Format(time.RFC3339),
	}
}

func (s *Service) GetCustomerBalance(ctx context.Context, customerID uuid.UUID) (CustomerBalanceDTO, error) {
	q := db.New(s.GetPool())
	cust, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID))
	if err != nil {
		return CustomerBalanceDTO{}, mapNotFound(err, ErrCustomerNotFound)
	}

	return CustomerBalanceDTO{
		CustomerID: customerID.String(),
		Balance:    formatMicro(cust.Balance),
		Currency:   cust.Currency,
		Ledger:     []LedgerDTO{},
	}, nil
}

func (s *Service) ExportCustomerLedgerCSV(ctx context.Context, customerID uuid.UUID, cursor int64, w io.Writer) (LedgerExportResult, error) {
	q := db.New(s.GetPool())
	if _, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID)); err != nil {
		return LedgerExportResult{}, err
	}

	limited := &limitedWriter{w: w, limit: ledgerExportMaxBytes}
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
			if limited.remaining() <= 0 {
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
				formatMicro(row.Amount),
				string(row.Type),
				row.IdempotencyHash.String,
				row.CreatedAt.Time.UTC().Format(time.RFC3339),
			}
			if err := cw.Write(record); err != nil {
				if limited.overflow() {
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
		if limited.remaining() <= 0 {
			truncated = true
			break
		}
	}

done:
	cw.Flush()
	if err := cw.Error(); err != nil {
		if limited.overflow() {
			truncated = true
		} else {
			return LedgerExportResult{}, err
		}
	}

	result := LedgerExportResult{
		Truncated: truncated,
		Bytes:     limited.bytesWritten(),
	}
	if truncated && lastID > 0 {
		result.NextCursor = lastID
	}
	return result, nil
}

func (s *Service) ListDisputes(ctx context.Context, customerFilter string, limit, offset int32) (adminapi.DisputeListResult, error) {
	if s.payment == nil {
		return adminapi.DisputeListResult{}, status.Error(codes.Unavailable, "payment service not configured")
	}
	resp, err := s.payment.ListDisputes(ctx, customerFilter, limit, offset)
	if err != nil {
		return adminapi.DisputeListResult{}, err
	}
	q := db.New(s.GetPool())
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

	rows := make([]adminapi.DisputeRowDTO, 0, len(resp.Disputes))
	for _, d := range resp.Disputes {
		item := adminapi.DisputeRowDTO{
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
	return adminapi.DisputeListResult{Disputes: rows, Total: resp.Total}, nil
}

type limitedWriter struct {
	w          io.Writer
	limit      int
	n          int
	overflowed bool
}

func (lw *limitedWriter) bytesWritten() int { return lw.n }

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.remaining() <= 0 {
		lw.overflowed = true
		return 0, errExportLimit
	}
	if len(p) > lw.remaining() {
		p = p[:lw.remaining()]
	}
	n, err := lw.w.Write(p)
	lw.n += n
	if n < len(p) || (err == nil && lw.remaining() == 0 && len(p) > 0) {
		lw.overflowed = true
	}
	if err != nil {
		return n, err
	}
	if lw.overflowed {
		return n, errExportLimit
	}
	return n, nil
}

func (lw *limitedWriter) remaining() int {
	return lw.limit - lw.n
}

func (lw *limitedWriter) overflow() bool {
	return lw.overflowed
}

var errExportLimit = fmt.Errorf("export byte limit reached")
