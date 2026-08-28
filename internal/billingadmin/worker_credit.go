package billingadmin

import (
	"context"
	"log/slog"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const creditBatchTimeout = 2 * time.Minute

type CreditScoringConfig struct {
	MinAgeDays         float64
	MatureAgeDays      float64
	MidTierPercent     int64
	MaturePercent      int64
	MaxCap             int64
	ReconLagThreshold  int64
	ReconLagPenaltyPct int64
}

type CreditHost interface {
	Pool() *pgxpool.Pool
	CreditScoringConfig() CreditScoringConfig
	UpdateOverdraft(ctx context.Context, customerID uuid.UUID, overdraft int64) error
}

type CreditWorker struct {
	host CreditHost
}

func NewCreditWorker(host CreditHost) *CreditWorker {
	return &CreditWorker{host: host}
}

func NewCreditScoringWorker(host CreditHost) *CreditWorker {
	return NewCreditWorker(host)
}

func (w *CreditWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.EvaluateAll(ctx); err != nil {
				slog.Error("credit scoring evaluation failed", "err", err)
			}
		}
	}
}

func (w *CreditWorker) EvaluateAll(ctx context.Context) error {
	opCtx, cancel := context.WithTimeout(ctx, creditBatchTimeout)
	defer cancel()

	queries := db.New(w.host.Pool())
	rows, err := queries.ListCustomersForScoring(opCtx)
	if err != nil {
		return err
	}

	lagByCustomer := make(map[uuid.UUID]int64)
	lagRows, err := queries.ListCustomerReconLagMicro(opCtx)
	if err != nil {
		slog.Error("failed to batch read recon lag", "err", err)
	} else {
		for _, row := range lagRows {
			lagByCustomer[uuid.UUID(row.CustomerID.Bytes)] = row.MaxLagMicro
		}
	}

	cfg := w.host.CreditScoringConfig()
	for _, r := range rows {
		customerID := uuid.UUID(r.ID.Bytes)
		reconLag := lagByCustomer[customerID]
		overdraft := calculateOverdraft(cfg, float64(r.AgeDays), r.TopupSum30d, reconLag)
		if err := w.host.UpdateOverdraft(opCtx, customerID, overdraft); err != nil {
			slog.Error("failed to update overdraft for customer", "customer_id", customerID, "err", err)
		}
	}

	return nil
}

func calculateOverdraft(cfg CreditScoringConfig, ageDays float64, topupSum int64, reconLagMicro int64) int64 {
	if ageDays < cfg.MinAgeDays {
		return 0
	}

	var overdraft int64
	if ageDays < cfg.MatureAgeDays {
		overdraft = topupSum * cfg.MidTierPercent / 100
	} else {
		overdraft = topupSum * cfg.MaturePercent / 100
	}

	if overdraft > cfg.MaxCap {
		overdraft = cfg.MaxCap
	}

	if cfg.ReconLagThreshold > 0 && reconLagMicro > cfg.ReconLagThreshold {
		penalty := cfg.ReconLagPenaltyPct
		if penalty < 0 {
			penalty = 0
		}
		if penalty > 100 {
			penalty = 100
		}
		overdraft = overdraft * (100 - penalty) / 100
	}

	return overdraft
}

func CalculateOverdraft(cfg CreditScoringConfig, ageDays float64, topupSum int64, reconLagMicro int64) int64 {
	return calculateOverdraft(cfg, ageDays, topupSum, reconLagMicro)
}
