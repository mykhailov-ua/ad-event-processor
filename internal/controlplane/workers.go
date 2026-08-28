package controlplane

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/nodeadmin"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/privacyadmin"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/internal/supply"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	workerBatchTimeout  = 2 * time.Minute
	workerDrainTimeout  = 30 * time.Second
	workerOutboxTimeout = 30 * time.Second

	incrementUsageMeterSQL = `
INSERT INTO billing.usage_meters (customer_id, meter, period, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (customer_id, meter, period) DO UPDATE
SET value = billing.usage_meters.value + EXCLUDED.value`
)

func workerContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return context.WithCancel(parent)
		}
		if remaining < timeout {
			timeout = remaining
		}
	}
	return context.WithTimeout(parent, timeout)
}

type TLSImpersonationWorker struct {
	svc *Service
}

func NewTLSImpersonationWorker(svc *Service) *TLSImpersonationWorker {
	return &TLSImpersonationWorker{svc: svc}
}

func (w *TLSImpersonationWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("TLSImpersonationWorker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.AnalyzeMismatches(ctx)
		}
	}
}

func (w *TLSImpersonationWorker) AnalyzeMismatches(ctx context.Context) {
	slog.Debug("TLSImpersonationWorker: analyzed TLS/UA mismatch metrics")
}

type (
	NodeCapacityScorerWorker        = nodeadmin.CapacityScorerWorker
	GlobalRegionTrafficScorerWorker = nodeadmin.GlobalTrafficScorerWorker
	NodeMetricsSnapshotWorker       = nodeadmin.MetricsSnapshotWorker
	NodeMetricsWorker               = nodeadmin.MetricsWorker
	ScheduleWorker                  = campaign.ScheduleWorker
	PacingControllerWorker          = campaign.PacingControllerWorker
	AutoscaleBudgetWorker           = campaign.AutoscaleBudgetWorker
	DeliveryOptimizerWorker         = campaign.DeliveryOptimizerWorker
	CampaignDrainWorker             = campaign.DrainWorker
	FloorOptimizerWorker            = rtbadmin.FloorOptimizerWorker
	BlacklistJanitor                = fraudadmin.BlacklistJanitor
	UsageDailyFlushWorker           = billingadmin.UsageDailyFlushWorker
	LedgerInvariantWorker           = billingadmin.LedgerInvariantWorker
	CreditScoringWorker             = billingadmin.CreditWorker
	VolumeMeterWorker               = billingadmin.VolumeMeterWorker
	ErasureWorker                   = privacyadmin.ErasureWorker
	ConsentRetentionWorker          = privacyadmin.ConsentRetentionWorker
	EventsRetentionWorker           = privacyadmin.EventsRetentionWorker
	SupplyAuditWorker               = supply.AuditWorker
	SupplyAuditReport               = supply.AuditReport
	SystemStateWorker               = platformadmin.SystemStateWorker
	NginxConfigWorker               = platformadmin.NginxWorker
	AuditExportWorker               = platformadmin.AuditExportWorker
	FraudModelSyncOrchestrator      = fraudadmin.MLSyncOrchestrator
	FraudModelSyncWorker            = fraudadmin.MLSyncWorker
	MLEvalMetricsWorker             = fraudadmin.MLEvalMetricsWorker
	MLShadowDeltaSnapshotWorker     = fraudadmin.MLShadowDeltaSnapshotWorker
	ReconWorker                     = reconciliation.ReconWorker
	RollupRow                       = billingadmin.RollupRow
)

const (
	VolumeMeterSourcePG = billingadmin.VolumeMeterSourcePG
	VolumeMeterSourceCH = billingadmin.VolumeMeterSourceCH
)

func NewScheduleWorker(svc *Service) *ScheduleWorker {
	return campaign.NewScheduleWorker(svc)
}

func NewPacingControllerWorker(svc *Service, syncWorkers []*domain.SyncWorker) *PacingControllerWorker {
	return campaign.NewPacingControllerWorker(svc, syncWorkers)
}

func NewAutoscaleBudgetWorker(svc *Service, syncWorkers []*domain.SyncWorker) *AutoscaleBudgetWorker {
	return campaign.NewAutoscaleBudgetWorker(svc, syncWorkers)
}

func NewDeliveryOptimizerWorker(svc *Service, syncWorkers []*domain.SyncWorker) *DeliveryOptimizerWorker {
	return campaign.NewDeliveryOptimizerWorker(svc, syncWorkers)
}

func NewCampaignDrainWorker(svc *Service) *CampaignDrainWorker {
	return campaign.NewDrainWorker(svc)
}

func NewFloorOptimizerWorker(svc *Service, interval time.Duration) *FloorOptimizerWorker {
	return rtbadmin.NewFloorOptimizerWorker(svc, interval)
}

func NewBlacklistJanitor(svc *Service, interval time.Duration) *BlacklistJanitor {
	return fraudadmin.NewBlacklistJanitor(svc, interval)
}

func NewUsageDailyFlushWorker(pool *pgxpool.Pool, interval time.Duration) *UsageDailyFlushWorker {
	return billingadmin.NewUsageDailyFlushWorker(pool, interval)
}

func NewLedgerInvariantWorker(pool *pgxpool.Pool, cfg *config.Config, notifier billingadmin.InvariantAlerter) *LedgerInvariantWorker {
	return billingadmin.NewLedgerInvariantWorker(pool, cfg, notifier)
}

func NewCreditScoringWorker(svc *Service) *CreditScoringWorker {
	return billingadmin.NewCreditScoringWorker(svc)
}

func NewVolumeMeterWorker(pool *pgxpool.Pool, clickhouseQuery *database.ClickHouseQuery, source string, interval time.Duration, postgresGate billingadmin.PostgresLowGate) *VolumeMeterWorker {
	return billingadmin.NewVolumeMeterWorker(pool, clickhouseQuery, source, interval, postgresGate)
}

func NewErasureWorker(svc *Service) *ErasureWorker {
	return privacyadmin.NewErasureWorker(svc)
}

func NewConsentRetentionWorker(svc *Service) *ConsentRetentionWorker {
	return privacyadmin.NewConsentRetentionWorker(svc)
}

func NewEventsRetentionWorker(pool *pgxpool.Pool, retentionDays int) *EventsRetentionWorker {
	return privacyadmin.NewEventsRetentionWorker(pool, retentionDays)
}

func NewSupplyAuditWorker(svc *Service) *SupplyAuditWorker {
	return supply.NewSupplyAuditWorker(svc)
}

func NewNginxConfigWorker(svc *Service, exportPath string) *NginxConfigWorker {
	return platformadmin.NewNginxConfigWorker(svc, exportPath)
}

func NewAuditExportWorker(svc *Service, exportPath string, retentionDays int) *AuditExportWorker {
	return platformadmin.NewAuditExportWorker(svc, exportPath, retentionDays)
}

func NewFraudModelSyncOrchestrator(svc *Service) *FraudModelSyncOrchestrator {
	return fraudadmin.NewFraudModelSyncOrchestrator(svc)
}

func NewFraudModelSyncWorker(svc *Service, interval time.Duration) *FraudModelSyncWorker {
	return fraudadmin.NewFraudModelSyncWorker(svc, interval)
}

func NewMLEvalMetricsWorker(svc *Service) *MLEvalMetricsWorker {
	return fraudadmin.NewMLEvalMetricsWorker(svc)
}

func NewMLShadowDeltaSnapshotWorker(svc *Service) *MLShadowDeltaSnapshotWorker {
	return fraudadmin.NewMLShadowDeltaSnapshotWorker(fraudMLShadowDeltaHost{svc: svc})
}

func NewSystemStateWorker(svc *Service) *SystemStateWorker {
	return platformadmin.NewSystemStateWorker(svc)
}

func NewReconWorker(svc *Service, interval time.Duration) *ReconWorker {
	return reconciliation.NewReconWorker(svc, interval)
}

func NewReconWorkerWithQuorum(svc *Service, interval, quorum time.Duration) *ReconWorker {
	return reconciliation.NewReconWorkerWithQuorum(svc, interval, quorum)
}

func ComputeWeightedUnitsFromRows(rows []RollupRow, campaignCustomers map[uuid.UUID]uuid.UUID) map[uuid.UUID]int64 {
	return billingadmin.ComputeWeightedUnitsFromRows(rows, campaignCustomers)
}

func (s *Service) AuditSupplyCompliance(ctx context.Context) (SupplyAuditReport, error) {
	return supply.AuditCompliance(ctx, s)
}
