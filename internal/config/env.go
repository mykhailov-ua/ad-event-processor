package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const ExpectedRedisShardCount = 4

type Config struct {
	ServerPort                      string
	ProcessorPort                   string
	ManagementPort                  string
	MetricsPort                     string
	DBDSN                           Secret
	PaymentDBDSN                    Secret
	RedisAddrs                      []string
	RedisSentinelAddrs              []string
	RedisMasterNames                []string
	RedisPassword                   Secret
	RedisStreamName                 string
	FraudStreamName                 string
	FraudConsumerLagSec             int
	H2IncompleteMax                 int
	RedisGroupName                  string
	RedisConsumerID                 string
	CHDSN                           Secret
	CHEnabled                       bool
	Env                             string
	TrustedProxies                  []string
	TokenSymmetricKey               Secret
	MaxRequestBodySize              int64
	ClickAmount                     int64
	ImpressionAmount                int64
	EventBatchSize                  int
	EventFlushMs                    int
	StatsFlushMs                    int
	MaxWorkers                      int
	CHMaxWorkers                    int
	ProcessorPGStreamMaxWorkers     int
	ProcessorCHStreamMaxWorkers     int
	ProcessorPGGateSlots            int
	ProcessorCHGateSlots            int
	ProcessorWeightEnabled          bool
	ProcessorWeightFloor            float64
	ProcessorWeightCeil             float64
	ProcessorWeightDrainPgWaitMs    int
	VendorTelemetryEnabled          bool
	VendorTelemetryIntervalSec      int
	VendorTelemetryTimeoutSec       int
	TelemetryOptIn                  bool
	TelemetryURL                    Secret
	TelemetryIntervalSec            int
	TelemetryHTTPTimeoutSec         int
	SyncWorkerMaxConcurrency        int
	LogRetentionDays                int
	DBTrackerMaxConns               int
	DBProcessorMaxConns             int
	DBMinConns                      int
	PgPoolSettleMaxConns            int
	VolumeMeterSource               string
	SettlementLanes                 int
	SettlementFlushMs               int
	ReconHYG30IntervalMs            int
	LedgerInvariantIntervalHours    int
	ReconForceRefill                bool
	CHMaxConns                      int
	CHQueryMaxConcurrency           int
	CHQueryTimeoutSec               int
	CHQuerySlowLogMS                int
	CHQueryMaxMemoryBytes           uint64
	CHQueryMaxExecSec               int
	CHSpoolDir                      string
	CHSpoolSegmentMB                int
	CHSpoolMaxSegments              int
	CHReadonlyDSN                   Secret
	CHRawRetentionDays              int
	CHEmergencyDropPercent          int
	CHRecompressPartsThreshold      int
	CHRecompressOffPeakStartUTC     int
	CHRecompressOffPeakEndUTC       int
	ProcessorStreamLagMaxSec        int
	TrackerPGFallback               bool
	WriteTimeoutMs                  int
	FilterTimeoutMs                 int
	FilterSlowMs                    int
	MetricsHistogramSampleMask      int
	AuditLogSampleMask              int
	AuditLedgerFlushSampleMask      int
	IdempotencyTTLHrs               int
	RateLimitPerMin                 int
	RateLimitWindowMs               int
	DuplicateTTLSec                 int
	TTCMinMs                        int
	TTCFailClosed                   bool
	CHBatchSize                     int
	CHFlushIntervalMs               int
	PIISaltVersion                  uint8
	PIISaltHex                      Secret
	PartitionPreCreateDays          int
	RegistrySyncIntervalMs          int
	BudgetSyncIntervalMs            int
	LedgerBatchFlushMs              int
	HttpReadHeaderTimeoutMs         int
	HttpReadTimeoutMs               int
	HttpWriteTimeoutMs              int
	HttpIdleTimeoutMs               int
	DefaultTokenDurationHrs         int
	StreamMaxLen                    int
	RedisStreamTrimIntervalMs       int
	RedisMaxActiveConns             int
	RetryInitialWaitMs              int
	RetryMaxWaitMs                  int
	MaxRetries                      int
	StreamMinIdleMs                 int
	Argon2Memory                    int
	Argon2Iterations                int
	Argon2Parallelism               int
	RedisPoolSize                   int
	RedisBreakerFailThreshold       int
	RedisBreakerHalfOpen            int
	RedisBreakerOpenTimeoutMs       int
	AdminAPIKey                     Secret
	InstallBootstrapToken           Secret
	AllowedOrigins                  []string
	PaymentWebhookPort              string
	PaymentInternalToken            Secret
	SettlementInternalToken         Secret
	StripeSecretKey                 Secret
	StripeWebhookSecret             Secret
	StripeCheckoutSuccessURL        string
	StripeCheckoutCancelURL         string
	CryptoWebhookSecret             Secret
	CryptoMinPaymentMicro           int64
	CryptoConfirmationDepth         int
	PaymentFinancialReconIntervalMs int

	SelfServeMaxActiveCampaigns int
	SelfServeMaxCreatesPerDay   int
	SelfServeBudgetMinMicro     int64
	SelfServeBudgetMaxMicro     int64
	SelfServeAPIKeyRPS          float64
	Management                  struct {
		RetentionDays               int
		CancellationFeePercent      float64
		ReconIntervalMs             int
		ReconSnapshotIntervalMs     int
		PacingIntervalMs            int
		RateLimitRPS                float64
		RateLimitBurst              int
		OpsAlertsEnabled            bool
		OpsAlertCooldownSec         int
		DrainStuckThresholdSec      int
		BlacklistJanitorEnabled     bool
		BlacklistJanitorIntervalSec int
		BlacklistAutoTTLHours       int
		BlacklistFraudTTLHours      int
		AlertmanagerWebhookEnabled  bool
		AlertmanagerWebhookToken    string
		OpsAlertOutboxStuckSec      int
		AuditExportPath             string
		AuditExportRetentionDays    int
		BillingExportPath           string
		SupplyExportPath            string
		AdminFanoutMaxConcurrency   int
		LowBalanceThresholdMicro    int64
		LowBalanceAlertEnabled      bool
	}
	Control struct {
		EnableAuth        bool
		EnableManagement  bool
		EnablePayment     bool
		EnableBilling     bool
		EnableNotifier    bool
		EnableMarginGuard bool
		EnableCostSync    bool
	}
	CampaignUpdateChannel   string
	RtbCatalogReloadChannel string

	RegistryStaleTTLSec          int
	RegistryPollMs               int
	CampaignUpdateBrokerFallback bool
	CampaignUpdateBrokerTopic    string
	RedisShard0OptionalStartup   bool
	CampaignReplicaPath          string

	AutoscaleHighCTRThreshold   float64
	AutoscaleMinImpressions     int64
	AutoscaleLowCTRThreshold    float64
	AutoscaleMinRemainingBudget int64
	AutoscaleShiftAmount        int64
	AutoscaleIntervalMs         int

	DeliveryOptimizerIntervalMs    int
	BidFloorLookbackHours          int
	BidFloorOptimizerLookbackHours int
	BidFloorOptimizerIntervalHours int
	BidFloorBucketMicro            int64
	BidFloorWinRateLow             float64
	BidFloorWinRateHigh            float64
	BidFloorAdjustPct              int
	BidFloorMinMicro               int64
	DealFloorRefreshIntervalMs     int

	PacingToleranceMargin float64

	MarginGuardIntervalSec         int
	MarginGuardDefaultThresholdBps int

	CreditScoringMinAgeDays         float64
	CreditScoringMatureAgeDays      float64
	CreditScoringMidTierPercent     int64
	CreditScoringMaturePercent      int64
	CreditScoringMaxCap             int64
	CreditScoringReconLagThreshold  int64
	CreditScoringReconLagPenaltyPct int64

	MABIntervalMs     int
	MABMinImpressions int64
	MABLookbackDays   int

	ConsentHMACSecret       Secret
	ConsentRetentionMonths  int
	ConsentUpdateChannel    string
	ErasureWorkerIntervalMs int
	EventsRetentionDays     int
	EventsHashIPAtInsert    bool

	Lifecycle struct {
		ShutdownTimeoutMs int
		DrainTimeoutMs    int
		WaitTimeoutMs     int
	}

	Logger struct {
		Dir                   string
		Shards                int
		FlushSizeKB           int
		RotateSizeMB          int
		RotateInterval        time.Duration
		LatencyLimit          time.Duration
		PersistQueueDepth     int
		PersistEnqueueTimeout time.Duration
	}

	Broker struct {
		URL                 string
		RedisURL            string
		Topic               string
		PartitionCount      int
		ShadowMode          bool
		CHIngestSource      string // "" = redis stream (default); "broker" = broker-primary, skips Redis _ch consumer
		MaxBytes            int
		TimeoutMs           int
		ReconcileIntervalMs int
		DivergenceThreshold uint64
	}

	RtbMode                        string
	RtbBudgetAuthority             string
	RtbClearingMode                string
	RtbSnapshotPath                string
	RtbHybridMaxRpsPerNode         int
	RtbReconcileIntervalMs         int
	RtbBudgetDivergenceMicro       int64
	RtbReconcileSampleSize         int
	RtbTargetingIndex              bool
	RtbPrebidIVT                   bool
	RtbExchangeMaxQPS              int
	RtbExchangeMaxBodyBytes        int64
	RtbExchangeNoBidMode           string
	RtbExchangeMultiImpMax         int
	RtbExchangeGzip                bool
	RtbExchangeDelivery            string
	RtbExchangeNURLTemplate        string
	TrackerTgClickBaseURL          string
	RtbExchangeSeatID              string
	RtbRegsPolicy                  string
	RtbCoppaPolicy                 string
	RtbBlocklistEnforce            bool
	RtbCatalogReloadSLOMs          int
	RtbDealOutcomeFlushMs          int
	CHJanitorEnabled               bool
	CHJanitorIntervalH             int
	CHRetentionDaysRtbDealOutcomes int
	CHRetentionDaysRtbExchangeLog  int

	IngressSchema string

	QuotaMode                 string
	LocalQuotaMode            string
	QuotaChunkSize            int64
	QuotaStrictThresholdMicro int64
	QuotaStrictExitMicro      int64
	QuotaRefillThresholdPct   int
	LocalQuotaRefillMaxShard  int
	QuotaAdaptiveFloorMicro   int64
	QuotaAdaptiveCeilingMicro int64
	BudgetDeltaTopic          string

	SlotMapReloadTopic            string
	SlotMapPollIntervalMs         int
	SlotMigrationEnabled          bool
	SlotMigrationIntervalMs       int
	MigrationFenceEnabled         bool
	SlotMigrationDualWriteEnabled bool
	SlotMigrationLagEpsilon       int64
	SlotMigrationLagThreshold     int64
	ElasticShardingEnabled        bool
	ShardOrchestratorEnabled      bool
	ShardOrchestratorIntervalMs   int
	TCPControlEnabled             bool
	TCPControlHMACSecret          Secret
	TCPControlBindAddr            string
	TCPControlAddr                string
	TCPTrackerAddrs               []string
	ManagementURL                 string

	LuaFastPathEnabled bool

	UDPControlEnabled  bool
	UDPFailClosed      bool
	UDPControlBindAddr string
	UDPTrackerBindAddr string
	UDPControlAddr     string
	UDPTrackerAddrs    []string
	UDPTrackerID       uint32
	UDPSyncIntervalMs  int
	UDPDefaultShardRPS uint64

	RegionCode          uint8
	MultiRegionEnabled  bool
	NodeID              string
	NodeRole            string
	NodeScoreWindowMin  int
	NodeScoreMinSamples int
	NodeWarmupSec       int
	ScoringWeightsJSON  string
	OpLeaseTimeoutSec   int
	OpLeaseMaxRenewals  int
	OpLeaseFencingDir   string

	GlobalSpendBatchMin        int
	GlobalSpendFlushIntervalMs int
	GlobalSpendMaxConcurrency  int

	RegionProxyAddr     string
	RegionProxyRedisURL string

	PgFailoverEnabled        bool
	PgFailoverCoordinator    bool
	PgPrimaryDSN             Secret
	PgStandbyDSN             Secret
	PgFailoverRedisURL       string
	PgFailoverHealthMs       int
	PgFailoverPollMs         int
	PgFailoverCoordMs        int
	PgFailoverLeaseSec       int
	PgFailoverFailThreshold  int
	PgPromoteCommand         string
	PgFailoverSnapshotSync   bool
	PgFailoverSyncPageSize   int
	PgFailoverAuditWindowSec int

	QuotaAutoRepair bool

	Notifier struct {
		WorkerIntervalMs           int
		WorkerBatchSize            int
		BreakerFailThreshold       int
		BreakerSuccessThreshold    int
		BreakerOpenTimeoutMs       int
		TelegramBotToken           Secret
		TelegramChatID             string
		SlackWebhookURL            Secret
		SMSProviderURL             string
		SMSAPIToken                Secret
		SMSDefaultRecipient        string
		SMTPHost                   string
		SMTPPort                   string
		SMTPUsername               string
		SMTPPassword               Secret
		SMTPSender                 string
		RetentionSentDays          int
		RetentionFailedDays        int
		RetentionIntervalHours     int
		InvoiceRecipient           string
		InvoiceProvider            string
		AdminBaseURL               string
		WorkerConcurrency          int
		DedupCooldownSec           int
		ClaimStaleSec              int
		GroupParallelism           int
		RateLimitPerMinute         int
		TelegramRateLimitPerMinute int
	}

	IVT struct {
		Enabled              bool
		ScanIntervalMs       int
		OutboxPendingLimit   int64
		WindowSec            int
		MinClicks            uint64
		MinImpressions       uint64
		ClickToImpRatio      float64
		MinIPsPerUA          uint64
		IntervalMinIntervals uint64
		IntervalMaxVariance  float64
	}

	FraudScoring struct {
		Enabled        bool
		ScanIntervalMs int
		BatchSize      int
		ModelPath      string
		Standalone     bool
	}

	GeoIP struct {
		DBPath              string
		StagingPath         string
		EditionID           string
		LicenseKey          string
		UpdaterEnabled      bool
		UpdateIntervalHours int
		WatcherIntervalSec  int
	}

	Billing struct {
		InvoiceWorkerEnabled bool
		PaymentProvider      string
		PaymentProviderKey   Secret
	}

	BillingInternalToken Secret
}

func (c *Config) MultiRegionCell() bool {
	return c != nil && c.MultiRegionEnabled && c.RegionCode != 0
}

func (c *Config) MultiRegionGlobal() bool {
	return c != nil && c.MultiRegionEnabled && c.RegionCode == 0
}

func (c *Config) BrokerEnabled() bool {
	return c != nil && c.Broker.URL != ""
}

func (c *Config) RedisSentinelEnabled() bool {
	return len(c.RedisSentinelAddrs) > 0
}

func (c *Config) ResolveRedisMasterNames() []string {
	if len(c.RedisMasterNames) > 0 {
		return c.RedisMasterNames
	}
	names := make([]string, len(c.RedisAddrs))
	for i := range c.RedisAddrs {
		names[i] = fmt.Sprintf("espx-shard-%d", i)
	}
	return names
}

func Load() (*Config, error) {
	appEnv := os.Getenv("ENV")
	cfg := &Config{
		ServerPort:                      os.Getenv("SERVER_PORT"),
		ProcessorPort:                   os.Getenv("PROCESSOR_PORT"),
		ManagementPort:                  os.Getenv("MANAGEMENT_PORT"),
		MetricsPort:                     os.Getenv("METRICS_PORT"),
		DBDSN:                           Secret(os.Getenv("DB_DSN")),
		PaymentDBDSN:                    Secret(os.Getenv("PAYMENT_DB_DSN")),
		RedisAddrs:                      trimCommaList(os.Getenv("REDIS_ADDRS")),
		RedisSentinelAddrs:              trimCommaList(os.Getenv("REDIS_SENTINEL_ADDRS")),
		RedisMasterNames:                trimCommaList(os.Getenv("REDIS_MASTER_NAMES")),
		RedisPassword:                   Secret(os.Getenv("REDIS_PASSWORD")),
		RedisStreamName:                 os.Getenv("REDIS_STREAM_NAME"),
		FraudStreamName:                 os.Getenv("FRAUD_STREAM_NAME"),
		FraudConsumerLagSec:             getEnvInt("FRAUD_CONSUMER_LAG_SEC", 30),
		H2IncompleteMax:                 getEnvInt("H2_INCOMPLETE_MAX", 3),
		RedisGroupName:                  os.Getenv("REDIS_GROUP_NAME"),
		RedisConsumerID:                 os.Getenv("REDIS_CONSUMER_ID"),
		EventBatchSize:                  getEnvInt("EVENT_BATCH_SIZE", 1000),
		EventFlushMs:                    getEnvInt("EVENT_FLUSH_MS", 500),
		StatsFlushMs:                    getEnvInt("STATS_FLUSH_MS", 5000),
		MaxWorkers:                      getEnvInt("MAX_WORKERS", 16),
		CHMaxWorkers:                    getEnvInt("CH_MAX_WORKERS", 1),
		ProcessorPGStreamMaxWorkers:     getEnvInt("PROCESSOR_PG_STREAM_MAX_WORKERS", 0),
		ProcessorCHStreamMaxWorkers:     getEnvInt("PROCESSOR_CH_STREAM_MAX_WORKERS", 0),
		ProcessorPGGateSlots:            getEnvInt("PROCESSOR_PG_GATE_SLOTS", 0),
		ProcessorCHGateSlots:            getEnvInt("PROCESSOR_CH_GATE_SLOTS", 0),
		ProcessorWeightEnabled:          getEnvBool("PROCESSOR_WEIGHT_ENABLED", false),
		ProcessorWeightFloor:            getEnvFloat("PROCESSOR_WEIGHT_FLOOR", 0.05),
		ProcessorWeightCeil:             getEnvFloat("PROCESSOR_WEIGHT_CEIL", 0.95),
		ProcessorWeightDrainPgWaitMs:    getEnvInt("PROCESSOR_WEIGHT_DRAIN_PG_WAIT_MS", 50),
		VendorTelemetryEnabled:          vendorTelemetryEnabled(appEnv),
		VendorTelemetryIntervalSec:      getEnvInt("VENDOR_TELEMETRY_INTERVAL_SEC", 60),
		VendorTelemetryTimeoutSec:       getEnvInt("VENDOR_TELEMETRY_TIMEOUT_SEC", 5),
		TelemetryOptIn:                  TelemetryOptInFromEnvDual(),
		TelemetryURL:                    Secret(strings.TrimSpace(TelemetryURLFromEnv())),
		TelemetryIntervalSec:            TelemetryIntervalSecFromEnv(),
		TelemetryHTTPTimeoutSec:         TelemetryHTTPTimeoutSecFromEnv(),
		SyncWorkerMaxConcurrency:        getEnvInt("SYNC_WORKER_MAX_CONCURRENCY", 32),
		LogRetentionDays:                getEnvInt("LOG_RETENTION_DAYS", 7),
		DBTrackerMaxConns:               getEnvInt("DB_TRACKER_MAX_CONNS", 4),
		DBProcessorMaxConns:             getEnvInt("DB_PROCESSOR_MAX_CONNS", 16),
		RedisMaxActiveConns:             getEnvIntDual("REDIS_MAX_ACTIVE_CONNS", "REDIS_MAX_ACTIVE", 2048),
		DBMinConns:                      getEnvInt("DB_MIN_CONNS", 2),
		PgPoolSettleMaxConns:            getEnvInt("PG_POOL_SETTLE_MAX_CONNS", 0),
		VolumeMeterSource:               envOrDefault("VOLUME_METER_SOURCE", "pg"),
		SettlementLanes:                 getEnvInt("SETTLEMENT_LANES", 0),
		SettlementFlushMs:               getEnvInt("SETTLEMENT_FLUSH_MS", 100),
		ReconHYG30IntervalMs:            getEnvInt("RECON_HYG30_INTERVAL_MS", 300_000),
		LedgerInvariantIntervalHours:    getEnvInt("LEDGER_INVARIANT_INTERVAL_HOURS", 24),
		ReconForceRefill:                getEnvBool("RECON_FORCE_REFILL", true),
		CHMaxConns:                      getEnvInt("CH_MAX_CONNS", 8),
		CHQueryMaxConcurrency:           getEnvInt("CH_QUERY_MAX_CONCURRENCY", 8),
		CHQueryTimeoutSec:               getEnvInt("CH_QUERY_TIMEOUT_SEC", 30),
		CHQuerySlowLogMS:                getEnvInt("CH_QUERY_SLOW_LOG_MS", 2000),
		CHQueryMaxMemoryBytes:           uint64(getEnvInt("CH_QUERY_MAX_MEMORY_BYTES", 0)),
		CHQueryMaxExecSec:               getEnvInt("CH_QUERY_MAX_EXEC_SEC", 0),
		CHSpoolDir:                      envOrDefault("CH_SPOOL_DIR", "/var/spool/espx/ch"),
		CHSpoolSegmentMB:                getEnvInt("CH_SPOOL_SEGMENT_MB", 512),
		CHSpoolMaxSegments:              getEnvInt("CH_SPOOL_MAX_SEGMENTS", 8),
		CHReadonlyDSN:                   Secret(envOrDefault("CH_READONLY_DSN", os.Getenv("CH_DSN"))),
		CHRawRetentionDays:              getEnvInt("CH_RAW_RETENTION_DAYS", 180),
		CHJanitorEnabled:                getEnvBool("CH_JANITOR_ENABLED", true),
		CHJanitorIntervalH:              getEnvInt("CH_JANITOR_INTERVAL_H", 24),
		CHRetentionDaysRtbDealOutcomes:  getEnvInt("CH_RETENTION_DAYS_RTB_DEAL_OUTCOMES", 90),
		CHRetentionDaysRtbExchangeLog:   getEnvInt("CH_RETENTION_DAYS_RTB_EXCHANGE_LOG", 30),
		CHEmergencyDropPercent:          getEnvInt("CH_EMERGENCY_DROP_PERCENT", 0),
		CHRecompressPartsThreshold:      getEnvInt("CH_RECOMPRESS_PARTS_THRESHOLD", 8),
		CHRecompressOffPeakStartUTC:     getEnvInt("CH_RECOMPRESS_OFFPEAK_START_UTC", 2),
		CHRecompressOffPeakEndUTC:       getEnvInt("CH_RECOMPRESS_OFFPEAK_END_UTC", 6),
		ProcessorStreamLagMaxSec:        getEnvInt("PROCESSOR_STREAM_LAG_MAX_SEC", 120),
		TrackerPGFallback:               getEnvBool("TRACKER_PG_FALLBACK", appEnv != "production"),
		WriteTimeoutMs:                  getEnvInt("WRITE_TIMEOUT_MS", 5000),
		FilterTimeoutMs:                 getEnvInt("FILTER_TIMEOUT_MS", 0),
		FilterSlowMs:                    getEnvInt("FILTER_SLOW_MS", 5),
		MetricsHistogramSampleMask:      getEnvInt("METRICS_HISTOGRAM_SAMPLE_MASK", 127),
		AuditLogSampleMask:              getEnvInt("AUDIT_LOG_SAMPLE_RATE", 127),
		AuditLedgerFlushSampleMask:      getEnvInt("AUDIT_LEDGER_FLUSH_SAMPLE_MASK", -1),
		IdempotencyTTLHrs:               getEnvInt("IDEMPOTENCY_TTL_HRS", 24),
		RateLimitPerMin:                 getEnvInt("RATE_LIMIT_PER_MIN", 100),
		RateLimitWindowMs:               getEnvInt("RATE_LIMIT_WINDOW_MS", 60000),
		MaxRequestBodySize:              getEnvInt64("MAX_REQUEST_BODY_SIZE", 1048576),
		DuplicateTTLSec:                 getEnvInt("DUPLICATE_TTL_SEC", 10),
		TTCMinMs:                        getEnvInt("TTC_MIN_MS", 300),
		TTCFailClosed:                   getEnvBool("TTC_FAIL_CLOSED", false),
		CHDSN:                           Secret(os.Getenv("CH_DSN")),
		CHEnabled:                       clickHouseEnabledFromEnv(),
		CHBatchSize:                     getEnvInt("CH_BATCH_SIZE", 50000),
		CHFlushIntervalMs:               getEnvInt("CH_FLUSH_INTERVAL_MS", 10000),
		PIISaltVersion:                  uint8(getEnvInt("PII_SALT_VERSION", 1)),
		PIISaltHex:                      Secret(os.Getenv("PII_SALT_HEX")),
		TokenSymmetricKey:               Secret(os.Getenv("TOKEN_SYMMETRIC_KEY")),
		PartitionPreCreateDays:          getEnvInt("PARTITION_PRECREATE_DAYS", 2),
		RegistrySyncIntervalMs:          getEnvInt("REGISTRY_SYNC_INTERVAL_MS", 60000),
		BudgetSyncIntervalMs:            getEnvInt("BUDGET_SYNC_INTERVAL_MS", 5000),
		LedgerBatchFlushMs:              getEnvInt("LEDGER_BATCH_FLUSH_MS", 10000),
		HttpReadHeaderTimeoutMs:         getEnvInt("HTTP_READ_HEADER_TIMEOUT_MS", 2000),
		HttpReadTimeoutMs:               getEnvInt("HTTP_READ_TIMEOUT_MS", 5000),
		HttpWriteTimeoutMs:              getEnvInt("HTTP_WRITE_TIMEOUT_MS", 10000),
		HttpIdleTimeoutMs:               getEnvInt("HTTP_IDLE_TIMEOUT_MS", 30000),
		DefaultTokenDurationHrs:         getEnvInt("DEFAULT_TOKEN_DURATION_HRS", 24),
		ClickAmount:                     getEnvMicro("CLICK_AMOUNT", 100_000),
		ImpressionAmount:                getEnvMicro("IMPRESSION_AMOUNT", 10_000),
		StreamMaxLen:                    getEnvIntDual("REDIS_STREAM_MAXLEN", "STREAM_MAX_LEN", 10000),
		RedisStreamTrimIntervalMs:       getEnvIntDual("REDIS_STREAM_TRIM_INTERVAL", "REDIS_STREAM_TRIM_INTERVAL_MS", 10000),
		RetryInitialWaitMs:              getEnvInt("RETRY_INITIAL_WAIT_MS", 100),
		RetryMaxWaitMs:                  getEnvInt("RETRY_MAX_WAIT_MS", 5000),
		MaxRetries:                      getEnvInt("MAX_RETRIES", 5),
		StreamMinIdleMs:                 getEnvInt("STREAM_MIN_IDLE_MS", 300000),
		Argon2Memory:                    getEnvInt("ARGON2_MEMORY", 65536),
		Argon2Iterations:                getEnvInt("ARGON2_ITERATIONS", 3),
		Argon2Parallelism:               getEnvInt("ARGON2_PARALLELISM", 4),
		RedisPoolSize:                   getEnvInt("REDIS_POOL_SIZE", 0),
		RedisBreakerFailThreshold:       getEnvInt("REDIS_BREAKER_FAIL_THRESHOLD", 150),
		RedisBreakerHalfOpen:            getEnvInt("REDIS_BREAKER_HALF_OPEN", 10),
		RedisBreakerOpenTimeoutMs:       getEnvInt("REDIS_BREAKER_OPEN_TIMEOUT_MS", 5000),
		AdminAPIKey:                     Secret(os.Getenv("ADMIN_API_KEY")),
		InstallBootstrapToken:           Secret(os.Getenv("INSTALL_BOOTSTRAP_TOKEN")),
		AllowedOrigins:                  strings.Split(os.Getenv("ALLOWED_ORIGINS"), ","),
		TrustedProxies:                  strings.Split(os.Getenv("TRUSTED_PROXIES"), ","),
		Env:                             appEnv,
		CampaignUpdateChannel:           os.Getenv("CAMPAIGN_UPDATE_CHANNEL"),
		RtbCatalogReloadChannel:         os.Getenv("RTB_CATALOG_RELOAD_CHANNEL"),
		RegistryStaleTTLSec:             getEnvInt("REGISTRY_STALE_TTL", 30),
		RegistryPollMs:                  getEnvInt("REGISTRY_POLL_MS", 5000),
		CampaignUpdateBrokerFallback:    getEnvBool("CAMPAIGN_UPDATE_BROKER_FALLBACK", appEnv == "production" || appEnv == "prod"),
		CampaignUpdateBrokerTopic:       envOrDefault("CAMPAIGN_UPDATE_BROKER_TOPIC", "campaigns:update"),
		RedisShard0OptionalStartup:      getEnvBool("REDIS_SHARD0_OPTIONAL_STARTUP", appEnv == "production" || appEnv == "prod"),
		CampaignReplicaPath:             envOrDefault("CAMPAIGN_REPLICA_PATH", "campaigns_replica.json"),
		AutoscaleHighCTRThreshold:       getEnvFloat("AUTOSCALE_HIGH_CTR_THRESHOLD", 0.015),
		AutoscaleMinImpressions:         getEnvInt64("AUTOSCALE_MIN_IMPRESSIONS", 100),
		AutoscaleLowCTRThreshold:        getEnvFloat("AUTOSCALE_LOW_CTR_THRESHOLD", 0.005),
		AutoscaleMinRemainingBudget:     getEnvMicro("AUTOSCALE_MIN_REMAINING_BUDGET", 20.0),
		AutoscaleShiftAmount:            getEnvMicro("AUTOSCALE_SHIFT_AMOUNT", 10.0),
		AutoscaleIntervalMs:             getEnvInt("AUTOSCALE_INTERVAL_MS", 0),
		DeliveryOptimizerIntervalMs:     getEnvInt("DELIVERY_OPTIMIZER_INTERVAL_MS", 0),
		BidFloorLookbackHours:           getEnvInt("BID_FLOOR_LOOKBACK_HOURS", 24),
		BidFloorOptimizerLookbackHours:  getEnvInt("BID_FLOOR_OPTIMIZER_LOOKBACK_HOURS", 168),
		BidFloorOptimizerIntervalHours:  getEnvInt("BID_FLOOR_OPTIMIZER_INTERVAL_HOURS", 168),
		BidFloorBucketMicro:             getEnvMicro("BID_FLOOR_BUCKET_MICRO", 10_000),
		BidFloorWinRateLow:              getEnvFloat("BID_FLOOR_WIN_RATE_LOW", 0.05),
		BidFloorWinRateHigh:             getEnvFloat("BID_FLOOR_WIN_RATE_HIGH", 0.25),
		BidFloorAdjustPct:               getEnvInt("BID_FLOOR_ADJUST_PCT", 10),
		BidFloorMinMicro:                getEnvMicro("BID_FLOOR_MIN_MICRO", 1000),
		DealFloorRefreshIntervalMs:      getEnvInt("DEAL_FLOOR_REFRESH_INTERVAL_MS", 60_000),
		PacingToleranceMargin:           getEnvFloat("PACING_TOLERANCE_MARGIN", 0.15),
		MarginGuardIntervalSec:          getEnvInt("MARGIN_GUARD_INTERVAL_SEC", 300),
		MarginGuardDefaultThresholdBps:  getEnvInt("MARGIN_GUARD_DEFAULT_THRESHOLD_BPS", 500),
		CreditScoringMinAgeDays:         getEnvFloat("CREDIT_SCORING_MIN_AGE_DAYS", 7.0),
		CreditScoringMatureAgeDays:      getEnvFloat("CREDIT_SCORING_MATURE_AGE_DAYS", 30.0),
		CreditScoringMidTierPercent:     getEnvInt64("CREDIT_SCORING_MID_TIER_PERCENT", 15),
		CreditScoringMaturePercent:      getEnvInt64("CREDIT_SCORING_MATURE_PERCENT", 30),
		CreditScoringMaxCap:             getEnvMicro("CREDIT_SCORING_MAX_CAP", 10000.0),
		CreditScoringReconLagThreshold:  getEnvMicro("CREDIT_SCORING_RECON_LAG_THRESHOLD_MICRO", 100.0),
		CreditScoringReconLagPenaltyPct: getEnvInt64("CREDIT_SCORING_RECON_LAG_PENALTY_PCT", 50),
		MABIntervalMs:                   getEnvInt("MAB_INTERVAL_MS", 900_000),
		MABMinImpressions:               getEnvInt64("MAB_MIN_IMPRESSIONS", 1000),
		MABLookbackDays:                 getEnvInt("MAB_LOOKBACK_DAYS", 90),
		ConsentHMACSecret:               Secret(os.Getenv("CONSENT_HMAC_SECRET")),
		ConsentRetentionMonths:          getEnvInt("CONSENT_RETENTION_MONTHS", 13),
		ConsentUpdateChannel:            envOrDefault("CONSENT_UPDATE_CHANNEL", "consent:update"),
		ErasureWorkerIntervalMs:         getEnvInt("ERASURE_WORKER_INTERVAL_MS", 60_000),
		EventsRetentionDays:             getEnvInt("EVENTS_RETENTION_DAYS", 90),
		EventsHashIPAtInsert:            getEnvBool("EVENTS_HASH_IP_AT_INSERT", false),
		PaymentWebhookPort:              os.Getenv("PAYMENT_WEBHOOK_PORT"),
		PaymentInternalToken:            Secret(os.Getenv("PAYMENT_INTERNAL_TOKEN")),
		SettlementInternalToken:         Secret(os.Getenv("SETTLEMENT_INTERNAL_TOKEN")),
		StripeSecretKey:                 Secret(os.Getenv("STRIPE_SECRET_KEY")),
		StripeWebhookSecret:             Secret(os.Getenv("STRIPE_WEBHOOK_SECRET")),
		StripeCheckoutSuccessURL:        os.Getenv("STRIPE_CHECKOUT_SUCCESS_URL"),
		StripeCheckoutCancelURL:         os.Getenv("STRIPE_CHECKOUT_CANCEL_URL"),
		CryptoWebhookSecret:             Secret(envOrDefault("CRYPTO_WEBHOOK_SECRET", "cryptosecret")),
		CryptoMinPaymentMicro:           getEnvMicro("CRYPTO_MIN_PAYMENT_MICRO", 10.0),
		CryptoConfirmationDepth:         getEnvInt("CRYPTO_CONFIRMATION_DEPTH", 12),
		PaymentFinancialReconIntervalMs: getEnvInt("PAYMENT_FINANCIAL_RECON_INTERVAL_MS", 0),
		SelfServeMaxActiveCampaigns:     getEnvInt("SELF_SERVE_MAX_ACTIVE_CAMPAIGNS", 500),
		SelfServeMaxCreatesPerDay:       getEnvInt("SELF_SERVE_MAX_CREATES_PER_DAY", 50),
		SelfServeBudgetMinMicro:         getEnvMicro("SELF_SERVE_BUDGET_MIN_MICRO", 1.0),
		SelfServeBudgetMaxMicro:         getEnvMicro("SELF_SERVE_BUDGET_MAX_MICRO", 1_000_000.0),
		SelfServeAPIKeyRPS:              getEnvFloat("SELF_SERVE_API_KEY_RPS", 30),
	}

	if err := loadIngestModules(cfg, appEnv); err != nil {
		return nil, err
	}
	loadEdgeModules(cfg)
	loadDatabaseModules(cfg)
	cfg.ManagementURL = os.Getenv("CONTROL_URL")
	if cfg.ManagementURL == "" {
		cfg.ManagementURL = os.Getenv("MANAGEMENT_URL")
	}
	if cfg.ManagementURL == "" && cfg.ManagementPort != "" {
		cfg.ManagementURL = "http://127.0.0.1:" + cfg.ManagementPort
	}

	loadControlplaneModules(cfg)
	loadManagementModules(cfg)

	if err := validateAndApplyDefaults(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) NotifierConfigured() bool {
	if c == nil {
		return false
	}
	return c.Notifier.TelegramBotToken != "" ||
		c.Notifier.TelegramChatID != "" ||
		c.Notifier.SlackWebhookURL != "" ||
		c.Notifier.SMTPHost != "" ||
		c.Notifier.SMTPSender != ""
}

func (c *Config) OpsAlertsEnabled() bool {
	if c == nil || !c.Management.OpsAlertsEnabled {
		return false
	}
	return c.opsAlertRecipient() != ""
}

func (c *Config) AlertmanagerWebhookEnabled() bool {
	if c == nil || !c.Management.AlertmanagerWebhookEnabled {
		return false
	}
	return c.opsAlertRecipient() != ""
}

func (c *Config) NotifierAPIEnabled() bool {
	return c.OpsAlertsEnabled() || c.AlertmanagerWebhookEnabled()
}

func (c *Config) opsAlertRecipient() string {
	if c.Notifier.TelegramChatID != "" {
		return c.Notifier.TelegramChatID
	}
	if c.Notifier.SlackWebhookURL != "" {
		return string(c.Notifier.SlackWebhookURL)
	}
	if c.Notifier.SMSDefaultRecipient != "" {
		return c.Notifier.SMSDefaultRecipient
	}
	if c.Notifier.SMTPSender != "" {
		return c.Notifier.SMTPSender
	}
	return ""
}

func (c *Config) IVTDetectorEnabled() bool {
	if c == nil || !c.IVT.Enabled {
		return false
	}
	return c.ClickHouseEnabled()
}

func (c *Config) FraudScoringEnabled() bool {
	return c != nil && c.FraudScoring.Enabled
}

func (c *Config) ProcessorPGStreamWorkers() int {
	if c == nil {
		return 16
	}
	if c.ProcessorPGStreamMaxWorkers > 0 {
		return c.ProcessorPGStreamMaxWorkers
	}
	if c.MaxWorkers > 0 {
		return c.MaxWorkers
	}
	return 16
}

func (c *Config) ProcessorCHStreamWorkers() int {
	if c == nil {
		return 1
	}
	if c.ProcessorCHStreamMaxWorkers > 0 {
		return c.ProcessorCHStreamMaxWorkers
	}
	if c.CHMaxWorkers > 0 {
		return c.CHMaxWorkers
	}
	return 1
}

func (c *Config) FraudScorerStandalone() bool {
	return c != nil && c.FraudScoring.Standalone
}

func (c *Config) ClickHouseEnabled() bool {
	return c != nil && c.CHEnabled && strings.TrimSpace(string(c.CHDSN)) != ""
}

func clickHouseEnabledFromEnv() bool {
	raw := strings.TrimSpace(os.Getenv("CH_ENABLED"))
	if raw == "" {
		return true
	}
	switch strings.ToLower(raw) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func telemetryOptInFromEnv() bool {
	return TelemetryOptInFromEnvDual()
}

func (c *Config) TelemetryInterval() time.Duration {
	if c == nil || c.TelemetryIntervalSec <= 0 {
		return time.Hour
	}
	return time.Duration(c.TelemetryIntervalSec) * time.Second
}

func (c *Config) TelemetryHTTPTimeout() time.Duration {
	if c == nil || c.TelemetryHTTPTimeoutSec <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.TelemetryHTTPTimeoutSec) * time.Second
}

func (c *Config) VolumeMeterFromPG() bool {
	if c == nil {
		return true
	}
	return c.VolumeMeterSource == "" || c.VolumeMeterSource == "pg"
}

func (c *Config) SettlementLaneCount() int {
	if c == nil {
		return runtime.GOMAXPROCS(0)
	}
	if c.SettlementLanes > 0 {
		return c.SettlementLanes
	}
	return runtime.GOMAXPROCS(0)
}

func (c *Config) ReconForceRefillEnabled() bool {
	return c != nil && c.ReconForceRefill
}

func (c *Config) PgPoolSettleConns(lanes int) int {
	if c == nil {
		return lanes + 2
	}
	if c.PgPoolSettleMaxConns > 0 {
		return c.PgPoolSettleMaxConns
	}
	return lanes + 2
}

func vendorTelemetryEnabled(appEnv string) bool {
	if v := os.Getenv("VENDOR_TELEMETRY_ENABLED"); v != "" {
		return getEnvBool("VENDOR_TELEMETRY_ENABLED", false)
	}
	if v := os.Getenv("ESPX_VENDOR_TELEMETRY"); v != "" {
		return getEnvBool("ESPX_VENDOR_TELEMETRY", false)
	}
	return appEnv == "production"
}
