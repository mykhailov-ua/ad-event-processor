package controlplane

import (
	"context"
	"log/slog"
	"os"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/identity"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/notify"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/privacyadmin"
	"ad-event-processor/internal/rtb"
	"ad-event-processor/internal/rtbadmin"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InProcessPaymentModule interface {
	SetSettlementAPI(api domain.PaymentSettlement)
	SetNotifierAPI(api notify.NotifierAPI)
	StartWorkers(ctx context.Context)
}

type InProcessBillingModule interface {
	ConfigureNotifier(api notify.NotifierAPI)
	StartWorkers(ctx context.Context)
}

type InProcessNotifierModule interface {
	StartWorkers(ctx context.Context)
}

type TCPControlPublisher interface {
	PublishSnapshot(ctx context.Context, routingEpoch int64, slotVersion int32)
}

type ControlServersStarter func(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, sharder domain.Sharder, numShards int) (TCPControlPublisher, func(), error)

type ServeOptions struct {
	Auth                *identity.AuthClient
	Billing             *ledger.BillingClient
	Payment             *payment.APIClient
	Notifier            *notify.Client
	BillingModule       InProcessBillingModule
	PaymentModule       InProcessPaymentModule
	NotifierModule      InProcessNotifierModule
	RtbBidShadeSim      rtbadmin.BidShadeSimulator
	StartControlServers ControlServersStarter
}

func (o ServeOptions) Monolith() bool {
	return o.Auth != nil && o.Billing != nil && o.Payment != nil && o.Notifier != nil
}

func startControlWorkers(
	ctx context.Context,
	cfg *config.Config,
	svc *Service,
	pool *pgxpool.Pool,
	postgresPools *database.PostgresPools,
	syncWorkers []*domain.SyncWorker,
) {
	reconInterval := time.Duration(cfg.Management.ReconIntervalMs) * time.Millisecond
	svc.StartReconWorker(reconInterval)
	slog.Info("started recon worker", "interval", reconInterval)

	volumeInterval := time.Hour
	if v := os.Getenv("VOLUME_METER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			volumeInterval = d
		}
	}
	if os.Getenv("VOLUME_METER_ENABLED") != "0" {
		meterSource := cfg.VolumeMeterSource
		var clickhouseQuery *database.ClickHouseQuery
		if meterSource == "ch" {
			clickhouseQuery = svc.ClickHouseQuery()
		}
		svc.StartBackgroundWorker(func() {
			billingadmin.NewVolumeMeterWorker(postgresPools.Settle, clickhouseQuery, meterSource, volumeInterval, svc.PostgresGate()).Start(ctx)
		})
		slog.Info("started volume meter worker", "interval", volumeInterval, "source", meterSource)
	}

	svc.StartBackgroundWorker(func() {
		billingadmin.NewLedgerInvariantWorker(postgresPools.Settle, cfg, nil).Start(ctx)
	})
	slog.Info("started ledger invariant worker", "interval_hours", cfg.LedgerInvariantIntervalHours)

	if cfg.BrokerEnabled() && (cfg.LocalQuotaMode == "shadow" || cfg.LocalQuotaMode == "live") {
		brokerRedisURL := cfg.Broker.RedisURL
		if brokerRedisURL == "" && len(cfg.RedisAddrs) > 0 {
			brokerRedisURL = database.BrokerRedisURL(cfg.RedisAddrs, string(cfg.RedisPassword))
		}
		budgetDeltaAgg := domain.NewBudgetDeltaAggregator()
		svc.SetBrokerDeltas(budgetDeltaAgg)
		deltaConsumer := rtb.NewBudgetDeltaConsumer(budgetDeltaAgg, rtb.BudgetDeltaConsumerConfig{
			BrokerAddr: cfg.Broker.URL,
			RedisURL:   brokerRedisURL,
			Topic:      cfg.BudgetDeltaTopic,
			Group:      cfg.RedisGroupName + "_control_budget_delta",
			MaxBytes:   uint32(cfg.Broker.MaxBytes),
			Timeout:    time.Duration(cfg.Broker.TimeoutMs) * time.Millisecond,
		})
		svc.StartBackgroundWorker(func() {
			deltaConsumer.Start(ctx)
		})
		slog.Info("management budget delta recon consumer enabled", "topic", cfg.BudgetDeltaTopic)
	}

	if cfg.QuotaMode == "shadow" || cfg.QuotaMode == "live" {
		svc.StartBackgroundWorker(func() {
			NewQuotaManager(svc).Start(ctx)
		})
		slog.Info("started quota manager", "mode", cfg.QuotaMode, "chunk_size", cfg.QuotaChunkSize, "refill_threshold_pct", cfg.QuotaRefillThresholdPct)
	}

	if cfg.DeliveryOptimizerIntervalMs > 0 {
		optimizerInterval := time.Duration(cfg.DeliveryOptimizerIntervalMs) * time.Millisecond
		svc.StartDeliveryOptimizerWorker(syncWorkers, optimizerInterval)
		slog.Info("started delivery optimizer worker", "interval", optimizerInterval, "mab_interval_ms", cfg.MABIntervalMs)
	}
	if cfg.BidFloorOptimizerIntervalHours > 0 {
		floorInterval := time.Duration(cfg.BidFloorOptimizerIntervalHours) * time.Hour
		svc.StartFloorOptimizerWorker(floorInterval)
		slog.Info("started floor optimizer worker", "interval", floorInterval)
	}

	pacingInterval := time.Duration(cfg.Management.PacingIntervalMs) * time.Millisecond
	svc.StartPacingController(syncWorkers, pacingInterval)
	slog.Info("started pacing controller", "interval", pacingInterval)

	if cfg.AutoscaleIntervalMs > 0 {
		autoscaleInterval := time.Duration(cfg.AutoscaleIntervalMs) * time.Millisecond
		svc.StartAutoscaleBudgetWorker(syncWorkers, autoscaleInterval)
		slog.Info("started autoscale budget worker", "interval", autoscaleInterval)
	}

	svc.StartAuditCleaner(platformadmin.Days(cfg.Management.RetentionDays))
	slog.Info("started audit cleaner", "retention_days", cfg.Management.RetentionDays)

	svc.StartBackgroundWorker(func() {
		privacyadmin.NewConsentRetentionWorker(svc).Start(ctx)
	})
	slog.Info("started consent retention worker", "retention_months", cfg.ConsentRetentionMonths)

	if cfg.EventsRetentionDays > 0 {
		svc.StartBackgroundWorker(func() {
			privacyadmin.NewEventsRetentionWorker(pool, cfg.EventsRetentionDays).Start(ctx)
		})
		slog.Info("started events retention worker", "retention_days", cfg.EventsRetentionDays)
	}

	if cfg.ErasureWorkerIntervalMs > 0 {
		erasureInterval := time.Duration(cfg.ErasureWorkerIntervalMs) * time.Millisecond
		svc.StartBackgroundWorker(func() {
			privacyadmin.NewErasureWorker(svc).Start(ctx, erasureInterval)
		})
		slog.Info("started privacy erasure worker", "interval", erasureInterval)
	}

	if cfg.Management.BlacklistJanitorEnabled {
		janitorInterval := time.Duration(cfg.Management.BlacklistJanitorIntervalSec) * time.Second
		svc.StartBlacklistJanitor(janitorInterval)
		slog.Info("started blacklist TTL janitor", "interval", janitorInterval)
	}

	svc.StartVendorTelemetryWorker(ctx)
	svc.StartProductTelemetryPulse(ctx)
	if cfg.TelemetryOptIn {
		slog.Info("product telemetry pulse enabled",
			"interval_sec", cfg.TelemetryIntervalSec,
			"url_configured", string(cfg.TelemetryURL) != "",
		)
	}
	if cfg.VendorTelemetryEnabled {
		slog.Info("vendor telemetry probes enabled",
			"interval_sec", cfg.VendorTelemetryIntervalSec,
			"timeout_sec", cfg.VendorTelemetryTimeoutSec,
		)
	}

	if exportPath := os.Getenv("NGINX_DENY_EXPORT_PATH"); exportPath != "" {
		nginxWorker := platformadmin.NewNginxConfigWorker(svc, exportPath)
		svc.StartBackgroundWorker(func() {
			nginxWorker.Start(ctx, time.Minute)
		})
		slog.Info("started nginx deny export worker", "path", exportPath)
	}

	if cfg.Management.AuditExportPath != "" {
		auditWorker := platformadmin.NewAuditExportWorker(svc, cfg.Management.AuditExportPath, cfg.Management.AuditExportRetentionDays)
		svc.StartBackgroundWorker(func() {
			auditWorker.Start(ctx, 24*time.Hour)
		})
		slog.Info("started audit export worker", "path", cfg.Management.AuditExportPath, "retention_days", cfg.Management.AuditExportRetentionDays)
	}
}
