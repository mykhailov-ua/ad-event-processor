package controlplane

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"ad-event-processor/internal/reportjob"

	"ad-event-processor/internal/billingadmin"
	campaignworker "ad-event-processor/internal/campaign/worker"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/fraud"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/internal/supply"

	"ad-event-processor/internal/automation"
	"ad-event-processor/pkg/domainhealth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	workerBatchTimeout  = 2 * time.Minute
	workerDrainTimeout  = 30 * time.Second
	workerOutboxTimeout = 30 * time.Second
)

type RollupRow = billingadmin.RollupRow

func ComputeWeightedUnitsFromRows(rows []RollupRow, campaignCustomers map[uuid.UUID]uuid.UUID) map[uuid.UUID]int64 {
	return billingadmin.ComputeWeightedUnitsFromRows(rows, campaignCustomers)
}

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

func (s *Service) fraudExplainScorer() (fraud.Scorer, error) {
	if s == nil || s.cfg == nil || !s.cfg.FraudScoring.ExplainLiveScore {
		return nil, errors.New("live fraud explain scoring disabled")
	}
	s.fraudExplainScorerMutex.Lock()
	defer s.fraudExplainScorerMutex.Unlock()
	if s.cachedFraudExplainScorer != nil {
		return s.cachedFraudExplainScorer, nil
	}
	if s.fraudExplainScorerErr != nil {
		return nil, s.fraudExplainScorerErr
	}
	modelPath := strings.TrimSpace(s.cfg.FraudScoring.ModelPath)
	if modelPath == "" {
		s.fraudExplainScorerErr = errors.New("fraud model path not configured")
		return nil, s.fraudExplainScorerErr
	}
	scorer, err := fraud.NewLGBMScorer(modelPath)
	if err != nil {
		s.fraudExplainScorerErr = err
		return nil, err
	}
	s.cachedFraudExplainScorer = scorer
	return scorer, nil
}

func (s *Service) StartMLShadowDeltaSnapshotWorker(ctx context.Context) {
	if s == nil {
		return
	}
	worker := fraudadmin.NewMLShadowDeltaSnapshotWorker(s)
	s.StartBackgroundWorker(func() { worker.Start(ctx) })
	slog.Info("ml shadow delta snapshot worker starting")
}

func (s *Service) RedisShardCount() int {
	if s == nil {
		return 0
	}
	return len(s.redisShards)
}

func (s *Service) RedisShard(shardID int) redis.UniversalClient {
	if s == nil || shardID < 0 || shardID >= len(s.redisShards) {
		return nil
	}
	return s.redisShards[shardID]
}

func (s *Service) ClickHouseOpContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return reports.ClickHouseQueryContext(ctx)
}

func (s *Service) SyncMLModelMetaOnShard(ctx context.Context, redisClient redis.UniversalClient, versionID, hash string, appliedAt int64) error {
	return shardadmin.SyncMLModelMetaOnShard(ctx, redisClient, versionID, hash, appliedAt)
}

func NewFraudModelSyncOrchestrator(svc *Service) *fraudadmin.MLSyncOrchestrator {
	return fraudadmin.NewFraudModelSyncOrchestrator(svc)
}

func NewReconWorkerWithQuorum(svc *Service, interval, quorum time.Duration) *reconciliation.ReconWorker {
	return reconciliation.NewReconWorkerWithQuorum(svc, interval, quorum)
}

func (s *Service) AuditSupplyCompliance(ctx context.Context) (supply.AuditReport, error) {
	return supply.AuditCompliance(ctx, s)
}

func (s *Service) StartReconWorker(interval time.Duration) {
	s.startWorker(func() {
		reconciliation.NewReconWorker(s, interval).Start(s.ctx)
	})
}

func (s *Service) StartAuditCleaner(retention platformadmin.Days) {
	s.startWorker(func() {
		s.RunAuditCleaner(s.ctx, retention)
	})
}

func (s *Service) StartBlacklistJanitor(interval time.Duration) {
	s.startWorker(func() {
		fraudadmin.NewBlacklistJanitor(s, interval).Start(s.ctx)
	})
}

func (s *Service) SetBrokerDeltas(reader reconciliation.BrokerPendingDeltaReader) {
	s.brokerDeltas = reader
}

func (s *Service) SetGlobalSpendReconciler(reconciler *reconciliation.GlobalSpendReconciler) {
	s.globalSpend = reconciler
}

func (s *Service) GlobalSpendReconciler() *reconciliation.GlobalSpendReconciler {
	if s == nil {
		return nil
	}
	return s.globalSpend
}

func (s *Service) StartPacingController(syncWorkers []*domain.SyncWorker, interval time.Duration) {
	s.startWorker(func() {
		campaignworker.NewPacingControllerWorker(s, syncWorkers).Start(s.ctx, interval)
	})
}

func (s *Service) StartAutoscaleBudgetWorker(syncWorkers []*domain.SyncWorker, interval time.Duration) {
	s.startWorker(func() {
		campaignworker.NewAutoscaleBudgetWorker(s, syncWorkers).Start(s.ctx, interval)
	})
}

func (s *Service) StartDeliveryOptimizerWorker(syncWorkers []*domain.SyncWorker, interval time.Duration) {
	s.startWorker(func() {
		campaignworker.NewDeliveryOptimizerWorker(s, syncWorkers).Start(s.ctx, interval)
	})
}

func startBuiltinServiceWorkers(s *Service, ctx context.Context, cfg *config.Config, pool *pgxpool.Pool) *Service {
	s.startWorker(func() {
		if cfg == nil {
			return
		}
		if cfg.MultiRegionCell() {
			NewRegionOutboxRelay(s).Start(ctx, 20*time.Millisecond)
			return
		}
		if !cfg.MultiRegionGlobal() {
			NewOutboxWorker(s).Start(ctx, 20*time.Millisecond)
		}
	})
	s.startWorker(func() {
		adapter := s.DedupAdapter(ctx)
		if adapter == nil {
			return
		}
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := adapter.RejectStaleProposals(ctx); err != nil && ctx.Err() == nil {
					slog.Warn("dedup proposal janitor failed", "err", err)
				}
			}
		}
	})
	s.startWorker(func() {
		campaignworker.NewDrainWorker(s).Start(ctx, 20*time.Millisecond)
	})
	if cfg != nil && cfg.MultiRegionEnabled {
		worker := NewNodeMetricsWorker(s)
		s.nodeMetrics = worker
		s.startWorker(func() {
			worker.Start(ctx)
		})
		snapshotWorker := NewNodeMetricsSnapshotWorker(s)
		s.startWorker(func() {
			snapshotWorker.Start(ctx)
		})
		store, err := NewScoringWeightsStore(ctx, pool, cfg)
		if err != nil {
			slog.Error("scoring weights config invalid", "err", err)
			s.cancel()
			return nil
		}
		s.scoringWeights = store
		s.startWorker(func() {
			store.Start(ctx, pool, cfg)
		})
		leaseWorker := NewOperationLeaseWorker(s)
		s.leaseWorker = leaseWorker
		s.startWorker(func() {
			leaseWorker.Start(ctx)
		})
	}
	if cfg != nil && cfg.MultiRegionCell() {
		scorerWorker := NewNodeCapacityScorerWorker(s)
		s.startWorker(func() {
			scorerWorker.Start(ctx)
		})
	}
	if cfg != nil && cfg.MultiRegionGlobal() {
		globalScorerWorker := NewGlobalRegionTrafficScorerWorker(s)
		s.startWorker(func() {
			globalScorerWorker.Start(ctx)
		})
	}
	s.startWorker(func() {
		billingadmin.NewCreditScoringWorker(s).Start(ctx, 24*time.Hour)
	})
	s.startWorker(func() {
		campaignworker.NewScheduleWorker(s).Start(ctx)
	})
	s.startWorker(func() {
		supply.NewSupplyAuditWorker(s).Start(ctx)
	})
	s.startWorker(func() {
		NewTLSImpersonationWorker(s).Start(ctx, 1*time.Hour)
	})
	s.startWorker(func() {
		platformadmin.NewSystemStateWorker(s).Start(ctx)
	})
	s.startWorker(func() {
		fraudadmin.NewMLEvalMetricsWorker(s).Start(ctx)
	})
	return s
}

func (s *Service) StartReportScheduleWorker(ctx context.Context) {
	if s == nil || s.pool == nil {
		return
	}
	runner := s.ReportJobRunner()
	if runner == nil || !runner.PgEnabled() {
		return
	}
	w := reportjob.NewReportScheduleWorker(s.pool, runner)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("report schedule worker starting")
}

func reportExportDirFromWire() string {
	return reportjob.DefaultReportExportDirPath()
}

func wireReportExportHooks() {
	labelFn := func(ctx context.Context) string {
		if user, ok := GetUser(ctx); ok {
			return user.UserID.String()
		}
		return ""
	}
	deploymentFn := func() string {
		if diag, ok := licenseWatcherDiagnostics(); ok {
			return diag.DeploymentID
		}
		return ""
	}
	reports.ExportActorLabel = labelFn
	reports.ExportDeploymentID = deploymentFn
	reportjob.ExportActorLabel = labelFn
	reportjob.ExportDeploymentID = deploymentFn
}

func (s *Service) StartAutomationWorker(ctx context.Context, intervalMinutes int) {
	if s == nil || s.cfg == nil || !s.cfg.Management.AutomationRulesEnabled {
		return
	}
	clickhouseQuery := s.ClickHouseQuery()
	if clickhouseQuery == nil {
		return
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	maxEvals := 50
	if s.cfg.Management.AutomationRulesMaxEvalsPerCustomerPerTick > 0 {
		maxEvals = s.cfg.Management.AutomationRulesMaxEvalsPerCustomerPerTick
	}
	host := automationHost{s}
	w := automation.NewWorker(s.pool, clickhouseQuery, automation.NewExecutor(host), interval, maxEvals)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("automation rules worker enabled", "interval", interval)
}

type automationHost struct {
	svc *Service
}

func (h automationHost) Pool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h automationHost) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return h.svc.PauseCampaign(ctx, campaignID, reason)
}

func (h automationHost) BlacklistPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	return h.svc.BlockCampaignPlacement(ctx, campaignID, placementID)
}

func (s *Service) EnqueueInviteEmail(ctx context.Context, email, url string) {
	if s == nil {
		return
	}
	title := "Team invitation"
	body := "Accept your invitation: " + url
	if s.notifierAPI != nil && s.cfg != nil {
		provider := s.cfg.Notifier.InvoiceProvider
		if provider == "" {
			provider = "SMTP"
		}
		if _, err := s.notifierAPI.SendNotification(ctx, provider, email, title, body); err != nil {
			slog.Warn("invite email enqueue failed", "email", email, "error", err)
		}
		return
	}
	slog.Info("invite email dry-run", "email", email, "url", url)
}

func (s *Service) AuditOwnerActivation(ctx context.Context, deploymentID, customerID, ownerUserID uuid.UUID) {
	if s == nil {
		return
	}
	s.AuditLog(ctx, nil, uuid.Nil, "OWNER_ACTIVATION", "deployment", &deploymentID, map[string]string{
		"customer_id":   customerID.String(),
		"owner_user_id": ownerUserID.String(),
	}, nil)
}

func (s *Service) handleMediaBuyerBudgetIncrease(ctx context.Context, locked db.Campaign, userID uuid.UUID, newLimit int64) error {
	return platformadmin.HandleMediaBuyerBudgetIncrease(ctx, s, locked, userID, newLimit)
}

func (s *Service) ReputationChecker() *domainhealth.ReputationChecker {
	if s == nil {
		return nil
	}
	if s.reputation != nil {
		return s.reputation
	}
	if s.cfg == nil || !s.cfg.Management.DomainReputationEnabled {
		return nil
	}
	s.reputation = domainhealth.NewReputationChecker(domainhealth.ReputationConfig{
		SafeBrowsingAPIKey: string(s.cfg.Management.SafeBrowsingAPIKey),
		FacebookToken:      string(s.cfg.Management.FacebookGraphAccessToken),
		FacebookGraphBase:  s.cfg.Management.FacebookGraphAPIBase,
	})
	return s.reputation
}

func (s *Service) SetCloudflareAPI(api platformadmin.CloudflareAPI) {
	if s == nil {
		return
	}
	s.cloudflare = api
}

func (s *Service) CloudflareClient() platformadmin.DomainCloudflareClient {
	if s == nil {
		return nil
	}
	if s.cloudflare != nil {
		return s.cloudflare
	}
	if s.cfg == nil {
		return nil
	}
	return platformadmin.NewCloudflareClient(string(s.cfg.Management.CloudflareAPIToken), s.cfg.Management.CloudflareAPIBase)
}

func (s *Service) RtbReconcileCHStats(ctx context.Context, requestID string, window time.Duration) (reconciliation.RtbReconcileCHStats, bool) {
	return reconciliation.RTBCHStats(ctx, s, requestID, window)
}
