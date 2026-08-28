package controlplane

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const usageExportBatchLimit = 500

type usageDailyExportRow struct {
	CustomerID   uuid.UUID
	CustomerName string
	CostCenter   string
	UsageDate    time.Time
	Meter        string
	Value        int64
}

func (s *Service) UpdateCustomerCostCenter(ctx context.Context, customerID uuid.UUID, costCenter string) (CustomerDTO, error) {
	normalized, err := billingadmin.NormalizeCostCenter(costCenter)
	if err != nil {
		return CustomerDTO{}, err
	}
	q := db.New(s.GetPool())
	if _, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID)); err != nil {
		return CustomerDTO{}, mapNotFound(err, ErrCustomerNotFound)
	}
	if _, err := q.UpdateCustomerCostCenter(ctx, db.UpdateCustomerCostCenterParams{
		ID:         domain.ToUUID(customerID),
		CostCenter: normalized,
	}); err != nil {
		return CustomerDTO{}, err
	}
	return s.GetCustomerDTO(ctx, customerID)
}

func (s *Service) enforceDeploymentTenantCap(ctx context.Context) error {
	if s == nil || s.GetPool() == nil {
		return nil
	}
	limits, state, ok := licenseDeploymentLimits()
	if !ok {
		return nil
	}
	if state == licensing.StateExpired || state == licensing.StateRevoked {
		return errValidation("license not active")
	}
	maxTenants := limits.MaxTenants
	if maxTenants == 0 || maxTenants >= 999999 {
		return nil
	}
	var count int64
	if err := s.GetPool().QueryRow(ctx, `SELECT COUNT(*) FROM customers`).Scan(&count); err != nil {
		return fmt.Errorf("count customers: %w", err)
	}
	if uint64(count) >= maxTenants {
		return ErrDeploymentTenantLimit
	}
	return nil
}

func (s *Service) ExportUsageDailyCSV(ctx context.Context, spec UsageExportSpec, w io.Writer) (UsageExportResult, error) {
	if s == nil || s.GetPool() == nil {
		return UsageExportResult{}, fmt.Errorf("service unavailable")
	}
	if spec.ToDate.Before(spec.FromDate) {
		return UsageExportResult{}, ErrInvalidTimeRange
	}

	limited := &limitedWriter{w: w, limit: exportChunkMaxBytes()}
	cw := csv.NewWriter(limited)
	if err := cw.Write([]string{"customer_id", "customer_name", "cost_center", "usage_date", "meter", "value"}); err != nil {
		return UsageExportResult{}, err
	}

	var (
		cursor    = spec.Cursor
		truncated bool
		lastRow   UsageExportCursor
		hasLast   bool
	)

	for {
		rows, err := s.listUsageDailyExportPage(ctx, spec, cursor, usageExportBatchLimit)
		if err != nil {
			return UsageExportResult{}, err
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			if limited.remaining() <= 0 {
				truncated = true
				goto done
			}
			record := []string{
				row.CustomerID.String(),
				row.CustomerName,
				row.CostCenter,
				row.UsageDate.Format("2006-01-02"),
				row.Meter,
				strconv.FormatInt(row.Value, 10),
			}
			if err := cw.Write(record); err != nil {
				if limited.overflow() {
					truncated = true
					goto done
				}
				return UsageExportResult{}, err
			}
			lastRow = UsageExportCursor{
				CustomerID: row.CustomerID,
				UsageDate:  row.UsageDate,
				Meter:      row.Meter,
				Valid:      true,
			}
			hasLast = true
		}

		if len(rows) < usageExportBatchLimit {
			break
		}
		cursor = lastRow
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
			return UsageExportResult{}, err
		}
	}

	result := UsageExportResult{
		Truncated: truncated,
		Bytes:     limited.bytesWritten(),
	}
	if truncated && hasLast {
		result.NextCursor = lastRow.Encode()
	}
	return result, nil
}

func (s *Service) listUsageDailyExportPage(ctx context.Context, spec UsageExportSpec, cursor UsageExportCursor, limit int) ([]usageDailyExportRow, error) {
	args := []any{spec.FromDate, spec.ToDate}
	where := []string{
		"ud.usage_date >= $1",
		"ud.usage_date <= $2",
	}
	argPos := 3

	if spec.CustomerID != nil {
		where = append(where, fmt.Sprintf("ud.customer_id = $%d", argPos))
		args = append(args, *spec.CustomerID)
		argPos++
	}
	if spec.CostCenter != "" {
		where = append(where, fmt.Sprintf("c.cost_center = $%d", argPos))
		args = append(args, spec.CostCenter)
		argPos++
	}
	if cursor.Valid {
		where = append(where, fmt.Sprintf("(ud.customer_id, ud.usage_date, ud.meter) > ($%d, $%d::date, $%d)", argPos, argPos+1, argPos+2))
		args = append(args, cursor.CustomerID, cursor.UsageDate, cursor.Meter)
		argPos += 3
	}

	query := fmt.Sprintf(`
		SELECT ud.customer_id, c.name, COALESCE(c.cost_center, ''), ud.usage_date, ud.meter, ud.value
		FROM billing.usage_daily ud
		JOIN customers c ON c.id = ud.customer_id
		WHERE %s
		ORDER BY ud.customer_id, ud.usage_date, ud.meter
		LIMIT $%d`, strings.Join(where, " AND "), argPos)
	args = append(args, limit)

	rows, err := s.GetPool().Query(ctx, query, args...)
	if err != nil {
		if isUndefinedRelation(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var out []usageDailyExportRow
	for rows.Next() {
		var row usageDailyExportRow
		var usageDate time.Time
		if err := rows.Scan(&row.CustomerID, &row.CustomerName, &row.CostCenter, &usageDate, &row.Meter, &row.Value); err != nil {
			return nil, err
		}
		row.UsageDate = usageDate
		out = append(out, row)
	}
	return out, rows.Err()
}

func isUndefinedRelation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}
