package billingadmin

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantCapHost interface {
	Pool() *pgxpool.Pool
	DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool)
	ErrValidation(msg string) error
}

type UsageExportHost interface {
	Pool() *pgxpool.Pool
	ExportChunkMaxBytes() int
}

const usageExportBatchLimit = 500

type usageDailyExportRow struct {
	CustomerID   uuid.UUID
	CustomerName string
	CostCenter   string
	UsageDate    time.Time
	Meter        string
	Value        int64
}

func EnforceDeploymentTenantCap(ctx context.Context, host TenantCapHost) error {
	if host == nil || host.Pool() == nil {
		return nil
	}
	limits, state, ok := host.DeploymentLimits()
	if !ok {
		return nil
	}
	if state == licensing.StateExpired || state == licensing.StateRevoked {
		return host.ErrValidation("license not active")
	}
	maxTenants := limits.MaxTenants
	if maxTenants == 0 || maxTenants >= 999999 {
		return nil
	}
	var count int64
	if err := host.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM customers`).Scan(&count); err != nil {
		return fmt.Errorf("count customers: %w", err)
	}
	if uint64(count) >= maxTenants {
		return ErrDeploymentTenantLimit
	}
	return nil
}

func ExportUsageDailyCSV(ctx context.Context, host UsageExportHost, spec UsageExportSpec, w io.Writer) (UsageExportResult, error) {
	if host == nil || host.Pool() == nil {
		return UsageExportResult{}, fmt.Errorf("service unavailable")
	}
	if spec.ToDate.Before(spec.FromDate) {
		return UsageExportResult{}, ErrInvalidTimeRange
	}

	limited := NewExportLimitedWriter(w, host.ExportChunkMaxBytes())
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
		rows, err := listUsageDailyExportPage(ctx, host.Pool(), spec, cursor, usageExportBatchLimit)
		if err != nil {
			return UsageExportResult{}, err
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			if limited.Remaining() <= 0 {
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
				if limited.Overflow() {
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
			return UsageExportResult{}, err
		}
	}

	result := UsageExportResult{
		Truncated: truncated,
		Bytes:     limited.BytesWritten(),
	}
	if truncated && hasLast {
		result.NextCursor = lastRow.Encode()
	}
	return result, nil
}

func listUsageDailyExportPage(ctx context.Context, pool *pgxpool.Pool, spec UsageExportSpec, cursor UsageExportCursor, limit int) ([]usageDailyExportRow, error) {
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

	rows, err := pool.Query(ctx, query, args...)
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

type exportLimitedWriter struct {
	w          io.Writer
	limit      int
	n          int
	overflowed bool
}

var ErrExportLimit = errors.New("export byte limit reached")

func NewExportLimitedWriter(w io.Writer, limit int) *exportLimitedWriter {
	return &exportLimitedWriter{w: w, limit: limit}
}

func (lw *exportLimitedWriter) BytesWritten() int { return lw.n }

func (lw *exportLimitedWriter) Remaining() int { return lw.limit - lw.n }

func (lw *exportLimitedWriter) Overflow() bool { return lw.overflowed }

func (lw *exportLimitedWriter) Write(p []byte) (int, error) {
	if lw.Remaining() <= 0 {
		lw.overflowed = true
		return 0, ErrExportLimit
	}
	if len(p) > lw.Remaining() {
		p = p[:lw.Remaining()]
	}
	n, err := lw.w.Write(p)
	lw.n += n
	if n < len(p) || (err == nil && lw.Remaining() == 0 && len(p) > 0) {
		lw.overflowed = true
	}
	if err != nil {
		return n, err
	}
	if lw.overflowed {
		return n, ErrExportLimit
	}
	return n, nil
}

func PatchCustomerCostCenter(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, costCenter string, mapNotFound func(error) error) error {
	normalized, err := NormalizeCostCenter(costCenter)
	if err != nil {
		return err
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE id = $1)`, domain.ToUUID(customerID)).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return mapNotFound(errors.New("not found"))
	}
	_, err = pool.Exec(ctx, `UPDATE customers SET cost_center = $1 WHERE id = $2`, normalized, domain.ToUUID(customerID))
	return err
}
