package platformadmin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func formatCustomerMicro(m int64) string {
	return money.FormatFixed2(m)
}

func customerListOrderClause(sortField, sortOrder string) string {
	dir := strings.ToUpper(strings.TrimSpace(sortOrder))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	switch strings.TrimSpace(sortField) {
	case "name":
		return "name " + dir
	default:
		return "created_at " + dir
	}
}

func (c *Customers) listCustomerRows(ctx context.Context, limit, offset int32, sortField, sortOrder string) ([]db.Customer, error) {
	pool := c.host.Pool()
	if pool == nil {
		return nil, errPlatformServiceUnavailable()
	}
	orderClause := customerListOrderClause(sortField, sortOrder)
	query := fmt.Sprintf(
		`SELECT id, name, balance, currency, cost_center, created_at, updated_at FROM customers ORDER BY %s LIMIT $1 OFFSET $2`,
		orderClause,
	)
	rows, err := pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []db.Customer
	for rows.Next() {
		var row db.Customer
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Balance,
			&row.Currency,
			&row.CostCenter,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (c *Customers) ListCustomers(ctx context.Context, limit, offset int32, sortField, sortOrder string) ([]CustomerDTO, int64, error) {
	if c == nil || c.host == nil || c.host.Pool() == nil {
		return nil, 0, errPlatformServiceUnavailable()
	}
	q := db.New(c.host.Pool())
	rows, total, err := coldpath.PaginatedQuery(
		func() (int64, error) { return q.CountCustomers(ctx) },
		func() ([]db.Customer, error) {
			return c.listCustomerRows(ctx, limit, offset, sortField, sortOrder)
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
			Balance:         formatCustomerMicro(r.Balance),
			Currency:        r.Currency,
			CostCenter:      r.CostCenter,
			ActiveCampaigns: st.ActiveCampaigns,
			TotalSpend:      formatCustomerMicro(st.TotalSpend),
			CreatedAt:       r.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:       r.UpdatedAt.Time.Format(time.RFC3339),
		}
	}), total, nil
}

func (c *Customers) GetCustomerDTO(ctx context.Context, id uuid.UUID) (CustomerDTO, error) {
	if c == nil || c.host == nil || c.host.Pool() == nil {
		return CustomerDTO{}, errPlatformServiceUnavailable()
	}
	q := db.New(c.host.Pool())
	r, err := q.GetCustomerByID(ctx, domain.ToUUID(id))
	if err != nil {
		return CustomerDTO{}, c.host.MapCustomerNotFound(err)
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
		Balance:         formatCustomerMicro(r.Balance),
		Currency:        r.Currency,
		CostCenter:      r.CostCenter,
		ActiveCampaigns: st.ActiveCampaigns,
		TotalSpend:      formatCustomerMicro(st.TotalSpend),
		CreatedAt:       r.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       r.UpdatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (c *Customers) UpdateCustomerCostCenter(ctx context.Context, customerID uuid.UUID, costCenter string) (CustomerDTO, error) {
	if c == nil || c.host == nil || c.host.Pool() == nil {
		return CustomerDTO{}, errPlatformServiceUnavailable()
	}
	normalized, err := billingadmin.NormalizeCostCenter(costCenter)
	if err != nil {
		return CustomerDTO{}, err
	}
	q := db.New(c.host.Pool())
	if _, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID)); err != nil {
		return CustomerDTO{}, c.host.MapCustomerNotFound(err)
	}
	if _, err := q.UpdateCustomerCostCenter(ctx, db.UpdateCustomerCostCenterParams{
		ID:         domain.ToUUID(customerID),
		CostCenter: normalized,
	}); err != nil {
		return CustomerDTO{}, err
	}
	return c.GetCustomerDTO(ctx, customerID)
}
