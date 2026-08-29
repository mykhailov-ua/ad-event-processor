package controlplane

import (
	"context"
	"errors"
	"fmt"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/governance"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/reconciliation"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrBudgetApprovalRequired   = governance.ErrBudgetApprovalRequired
	ErrBudgetApprovalAutoDenied = governance.ErrBudgetApprovalAutoDenied
)

var NewQuotaManager = governance.NewQuotaManager

var (
	_ governance.Host                = (*Service)(nil)
	_ reconciliation.ReconInfraHost  = (*Service)(nil)
	_ reconciliation.ReconOpsHost    = (*Service)(nil)
	_ reconciliation.ReconRepairHost = (*Service)(nil)
	_ reconciliation.Host            = (*Service)(nil)
)

func (s *Service) Config() *config.Config {
	if s == nil {
		return nil
	}
	return s.cfg
}

func (s *Service) SetSettlePool(pool *pgxpool.Pool) {
	if s == nil {
		return
	}
	s.settlementPostgresPool = pool
}

func (s *Service) SettlementPool() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	if s.settlementPostgresPool != nil {
		return s.settlementPostgresPool
	}
	return s.GetPool()
}

func (s *Service) PaymentQueryPool() reconciliation.PaymentQueryer {
	if s == nil {
		return nil
	}
	if s.paymentPool != nil {
		return s.paymentPool
	}
	return s.GetPool()
}

func (s *Service) Sharder() domain.Sharder {
	if s == nil {
		return nil
	}
	return s.sharder
}

func (s *Service) Alerter() reconciliation.Alerter {
	if s == nil {
		return nil
	}
	return s.alerter
}

func (s *Service) BrokerDeltas() reconciliation.BrokerPendingDeltaReader {
	if s == nil {
		return nil
	}
	return s.brokerDeltas
}

func (s *Service) InvalidServiceFilterErr() error {
	return ErrInvalidServiceFilter
}

func (s *Service) checkMediaBuyerBudgetCap(ctx context.Context, userID, campaignID uuid.UUID, newLimit int64) error {
	return governance.CheckMediaBuyerBudgetCap(ctx, s.GetPool(), userID, campaignID, newLimit)
}

func (s *Service) CheckMediaBuyerBudgetCap(ctx context.Context, userID, campaignID uuid.UUID, newLimit int64) error {
	return s.checkMediaBuyerBudgetCap(ctx, userID, campaignID, newLimit)
}

func (s *Service) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]opsadmin.ReconRunDTO, int64, error) {
	return reconciliation.ListRuns(ctx, s, service, limit, offset)
}

func (s *Service) applyRegionSpendSyncBatch(ctx context.Context, batchDedupKey string, payload []byte) error {
	reconciler := s.GlobalSpendReconciler()
	if reconciler == nil {
		return nil
	}
	return reconciler.ApplyRegionSpendSyncBatch(ctx, batchDedupKey, payload)
}

func (s *Service) ForceRefillCampaignFromPG(ctx context.Context, campaignID uuid.UUID, currentSpend int64) error {
	if s == nil || s.GetPool() == nil {
		return errors.New("service unavailable")
	}
	var budgetLimit int64
	err := s.GetPool().QueryRow(ctx, `SELECT budget_limit FROM campaigns WHERE id = $1`, domain.ToUUID(campaignID)).Scan(&budgetLimit)
	if err != nil {
		return err
	}
	remaining := budgetLimit - currentSpend
	if remaining < 0 {
		remaining = 0
	}
	redisClient := s.RedisClientForCampaign(campaignID)
	if redisClient == nil {
		return fmt.Errorf("no redis shard for campaign %s", campaignID)
	}
	return redisClient.Set(ctx, domain.BudgetCampaignKey(campaignID), remaining, 0).Err()
}

func (s *Service) RedisClientForCampaign(campaignID uuid.UUID) redis.UniversalClient {
	return s.redisClientForCampaign(campaignID)
}

func (s *Service) RunStuckDrainCheck(ctx context.Context) {
	if s == nil {
		return
	}
	s.CheckStuckDrainJobs(ctx)
}
