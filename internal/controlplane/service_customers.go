package controlplane

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"espx/internal/domain"
	"espx/internal/domain/db"
	"espx/pkg/coldpath"
	"espx/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CustomerDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Balance         string `json:"balance"`
	Currency        string `json:"currency"`
	ActiveCampaigns int64  `json:"active_campaigns"`
	TotalSpend      string `json:"total_spend"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type LedgerDTO struct {
	ID              int64  `json:"id"`
	CustomerID      string `json:"customer_id"`
	CampaignID      string `json:"campaign_id,omitempty"`
	Amount          string `json:"amount"`
	Type            string `json:"type"`
	IdempotencyHash string `json:"idempotency_hash,omitempty"`
	CreatedAt       string `json:"created_at"`
}

const (
	ledgerExportMaxBytes   = 10 * 1024 * 1024
	ledgerExportBatchLimit = 500
)

type CustomerBalanceDTO struct {
	CustomerID string      `json:"customer_id"`
	Balance    string      `json:"balance"`
	Currency   string      `json:"currency"`
	Ledger     []LedgerDTO `json:"ledger"`
}

type LedgerExportResult struct {
	NextCursor int64
	Truncated  bool
	Bytes      int
}

func ledgerToDTO(r db.BalanceLedger) LedgerDTO {
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

func formatMicro(m int64) string {
	return money.FormatFixed2(m)
}

func (s *Service) ListCustomers(ctx context.Context, limit, offset int32) ([]CustomerDTO, int64, error) {
	q := db.New(s.GetPool())
	total, err := q.CountCustomers(ctx)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []CustomerDTO{}, 0, nil
	}

	rows, err := q.ListCustomers(ctx, db.ListCustomersParams{Limit: limit, Offset: offset})
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

	statsMap := coldpath.KeyBy(stats, func(st db.GetCustomerStatsRow) (uuid.UUID, bool) {
		if st.CustomerID.Valid {
			return uuid.UUID(st.CustomerID.Bytes), true
		}
		return uuid.Nil, false
	})

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
		ledgerToDTO,
	)
}

func (s *Service) GetCustomerBalance(ctx context.Context, customerID uuid.UUID) (CustomerBalanceDTO, error) {
	q := db.New(s.GetPool())
	cust, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID))
	if err != nil {
		return CustomerBalanceDTO{}, mapNotFound(err, ErrCustomerNotFound)
	}

	rows, err := q.ListCustomerLedgerByIDDesc(ctx, domain.ToUUID(customerID))
	if err != nil {
		return CustomerBalanceDTO{}, err
	}

	ledger := make([]LedgerDTO, 0, len(rows))
	for _, row := range rows {
		ledger = append(ledger, ledgerToDTO(row))
	}

	return CustomerBalanceDTO{
		CustomerID: customerID.String(),
		Balance:    formatMicro(cust.Balance),
		Currency:   cust.Currency,
		Ledger:     ledger,
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
