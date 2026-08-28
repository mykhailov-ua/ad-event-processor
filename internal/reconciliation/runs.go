package reconciliation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/opsadmin"

	"github.com/jackc/pgx/v5/pgtype"
)

func ListRuns(ctx context.Context, host Host, service string, limit, offset int32) ([]opsadmin.ReconRunDTO, int64, error) {
	switch service {
	case "", "all":
		service = "all"
	case "management", "payment":
	default:
		return nil, 0, fmt.Errorf("%w: %s", host.InvalidServiceFilterErr(), service)
	}

	var runs []opsadmin.ReconRunDTO
	var total int64

	if service == "all" || service == "management" {
		q := db.New(host.Pool())
		count, err := q.CountManagementReconRuns(ctx)
		if err != nil {
			return nil, 0, err
		}
		total += count
		if count > 0 {
			rows, err := q.ListManagementReconRuns(ctx, db.ListManagementReconRunsParams{
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return nil, 0, err
			}
			for _, row := range rows {
				totalDelta := row.TotalDelta
				campaignsChecked := row.CampaignsChecked
				discrepanciesFound := row.DiscrepanciesFound
				dto := opsadmin.ReconRunDTO{
					Service:            "management",
					ID:                 row.ID,
					PeriodStart:        row.PeriodStart.Time.UTC().Format(time.RFC3339),
					PeriodEnd:          row.PeriodEnd.Time.UTC().Format(time.RFC3339),
					Status:             row.Status,
					TotalDelta:         &totalDelta,
					CampaignsChecked:   &campaignsChecked,
					DiscrepanciesFound: &discrepanciesFound,
					CreatedAt:          row.CreatedAt.Time.UTC().Format(time.RFC3339),
				}
				if row.CompletedAt.Valid {
					dto.CompletedAt = row.CompletedAt.Time.UTC().Format(time.RFC3339)
				}
				runs = append(runs, dto)
			}
		}
	}

	if service == "all" || service == "payment" {
		paymentRuns, paymentTotal, err := listPaymentReconRuns(ctx, host, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		runs = append(runs, paymentRuns...)
		total += paymentTotal
	}

	if service == "all" {
		sort.Slice(runs, func(i, j int) bool {
			return runs[i].CreatedAt > runs[j].CreatedAt
		})
		if int32(len(runs)) > limit {
			runs = runs[:limit]
		}
	}

	return runs, total, nil
}

func listPaymentReconRuns(ctx context.Context, host Host, limit, offset int32) ([]opsadmin.ReconRunDTO, int64, error) {
	pool := host.PaymentQueryPool()

	var total int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM payment.financial_recon_runs`).Scan(&total); err != nil {
		if isMissingPaymentSchema(err) {
			return []opsadmin.ReconRunDTO{}, 0, nil
		}
		return nil, 0, err
	}
	if total == 0 {
		return []opsadmin.ReconRunDTO{}, 0, nil
	}

	rows, err := pool.Query(ctx, `
		SELECT id, period_start, period_end, status::text, findings_count, intents_checked,
		 error_message, created_at, completed_at
		FROM payment.financial_recon_runs
		ORDER BY id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		if isMissingPaymentSchema(err) {
			return []opsadmin.ReconRunDTO{}, 0, nil
		}
		return nil, 0, err
	}
	defer rows.Close()

	var runs []opsadmin.ReconRunDTO
	for rows.Next() {
		var (
			id             int64
			periodStart    pgtype.Timestamptz
			periodEnd      pgtype.Timestamptz
			status         string
			findingsCount  int32
			intentsChecked int32
			errorMessage   pgtype.Text
			createdAt      pgtype.Timestamptz
			completedAt    pgtype.Timestamptz
		)
		if err := rows.Scan(&id, &periodStart, &periodEnd, &status, &findingsCount, &intentsChecked, &errorMessage, &createdAt, &completedAt); err != nil {
			return nil, 0, err
		}
		dto := opsadmin.ReconRunDTO{
			Service:        "payment",
			ID:             id,
			PeriodStart:    periodStart.Time.UTC().Format(time.RFC3339),
			PeriodEnd:      periodEnd.Time.UTC().Format(time.RFC3339),
			Status:         status,
			FindingsCount:  &findingsCount,
			IntentsChecked: &intentsChecked,
			CreatedAt:      createdAt.Time.UTC().Format(time.RFC3339),
		}
		if errorMessage.Valid {
			dto.ErrorMessage = errorMessage.String
		}
		if completedAt.Valid {
			dto.CompletedAt = completedAt.Time.UTC().Format(time.RFC3339)
		}
		runs = append(runs, dto)
	}
	return runs, total, rows.Err()
}

func isMissingPaymentSchema(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, `schema "payment"`)
}
