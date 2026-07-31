package config

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
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
	AuthServerPort                  string
	AuthMetricsPort                 string
	AuthGRPCEnabled                 bool
	BillingGRPCEnabled              bool
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
	AllowedOrigins                  []string
	PaymentServerPort               string
	PaymentServerHost               string
	PaymentMetricsPort              string
	PaymentWebhookPort              string
	PaymentGRPCEnabled              bool
	NotifierGRPCEnabled             bool
	SettlementServerPort            string
	SettlementServerHost            string
	SettlementGRPCEnabled           bool
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
		MaxBytes            int
		TimeoutMs           int
		ReconcileIntervalMs int
		DivergenceThreshold uint64
	}

	RtbMode                  string
	RtbBudgetAuthority       string
	RtbClearingMode          string
	RtbSnapshotPath          string
	RtbHybridMaxRpsPerNode   int
	RtbReconcileIntervalMs   int
	RtbBudgetDivergenceMicro int64
	RtbReconcileSampleSize   int
	RtbTargetingIndex        bool
	RtbPrebidIVT             bool

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
	TCPMgmtBindAddr               string
	TCPMgmtAddr                   string
	TCPTrackerAddrs               []string
	ManagementURL                 string

	LuaFastPathEnabled bool

	UDPControlEnabled  bool
	UDPFailClosed      bool
	UDPMgmtBindAddr    string
	UDPTrackerBindAddr string
	UDPMgmtAddr        string
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
		ServerHost                 string
		Port                       string
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
		MetricsPort                string
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
		Port                 string
		ServerHost           string
		MetricsPort          string
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
		AuthServerPort:                  os.Getenv("AUTH_SERVER_PORT"),
		AuthGRPCEnabled:                 os.Getenv("AUTH_GRPC_ENABLED") != "0",
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
		StreamMaxLen:                    getEnvInt("STREAM_MAX_LEN", 100000),
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
		AllowedOrigins:                  strings.Split(os.Getenv("ALLOWED_ORIGINS"), ","),
		TrustedProxies:                  strings.Split(os.Getenv("TRUSTED_PROXIES"), ","),
		Env:                             appEnv,
		AuthMetricsPort:                 os.Getenv("AUTH_METRICS_PORT"),
		CampaignUpdateChannel:           os.Getenv("CAMPAIGN_UPDATE_CHANNEL"),
		RtbCatalogReloadChannel:         os.Getenv("RTB_CATALOG_RELOAD_CHANNEL"),
		RegistryStaleTTLSec:             getEnvInt("REGISTRY_STALE_TTL", 30),
		RegistryPollMs:                  getEnvInt("REGISTRY_POLL_MS", 5000),
		CampaignUpdateBrokerFallback:    getEnvBool("CAMPAIGN_UPDATE_BROKER_FALLBACK", false),
		CampaignUpdateBrokerTopic:       envOrDefault("CAMPAIGN_UPDATE_BROKER_TOPIC", "campaigns:update"),
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
		PaymentServerPort:               os.Getenv("PAYMENT_SERVER_PORT"),
		PaymentServerHost:               os.Getenv("PAYMENT_SERVER_HOST"),
		PaymentMetricsPort:              os.Getenv("PAYMENT_METRICS_PORT"),
		PaymentWebhookPort:              os.Getenv("PAYMENT_WEBHOOK_PORT"),
		PaymentGRPCEnabled:              os.Getenv("PAYMENT_GRPC_ENABLED") != "0",
		SettlementServerPort:            os.Getenv("SETTLEMENT_SERVER_PORT"),
		SettlementServerHost:            os.Getenv("SETTLEMENT_SERVER_HOST"),
		SettlementGRPCEnabled:           os.Getenv("SETTLEMENT_GRPC_ENABLED") != "0",
		BillingGRPCEnabled:              os.Getenv("BILLING_GRPC_ENABLED") != "0",
		NotifierGRPCEnabled:             os.Getenv("NOTIFIER_GRPC_ENABLED") != "0",
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
	cfg.ElasticShardingEnabled = getEnvBool("ELASTIC_SHARDING_ENABLED", false)
	cfg.ShardOrchestratorEnabled = getEnvBool("SHARD_ORCHESTRATOR_ENABLED", false)
	cfg.ShardOrchestratorIntervalMs = getEnvInt("SHARD_ORCHESTRATOR_INTERVAL_MS", 10000)
	cfg.TCPControlEnabled = getEnvBool("TCP_CONTROL_ENABLED", false)
	cfg.TCPControlHMACSecret = Secret(os.Getenv("TCP_CONTROL_HMAC_SECRET"))
	cfg.TCPMgmtBindAddr = os.Getenv("TCP_MGMT_BIND_ADDR")
	if cfg.TCPMgmtBindAddr == "" {
		cfg.TCPMgmtBindAddr = ":8192"
	}
	cfg.TCPMgmtAddr = os.Getenv("TCP_MGMT_ADDR")
	if cfg.TCPMgmtAddr == "" {
		cfg.TCPMgmtAddr = "127.0.0.1:8192"
	}
	if addrs := os.Getenv("TCP_TRACKER_ADDRS"); addrs != "" {
		cfg.TCPTrackerAddrs = strings.Split(addrs, ",")
	}
	cfg.LuaFastPathEnabled = getEnvBool("LUA_FAST_PATH_ENABLED", true)
	cfg.UDPControlEnabled = getEnvBool("UDP_CONTROL_ENABLED", false)
	cfg.UDPFailClosed = getEnvBool("UDP_FAIL_CLOSED", true)
	cfg.UDPMgmtBindAddr = os.Getenv("UDP_MGMT_BIND_ADDR")
	if cfg.UDPMgmtBindAddr == "" {
		cfg.UDPMgmtBindAddr = ":8190"
	}
	cfg.UDPTrackerBindAddr = os.Getenv("UDP_TRACKER_BIND_ADDR")
	if cfg.UDPTrackerBindAddr == "" {
		cfg.UDPTrackerBindAddr = ":8191"
	}
	cfg.UDPMgmtAddr = os.Getenv("UDP_MGMT_ADDR")
	if cfg.UDPMgmtAddr == "" {
		cfg.UDPMgmtAddr = "127.0.0.1:8190"
	}
	if addrs := os.Getenv("UDP_TRACKER_ADDRS"); addrs != "" {
		cfg.UDPTrackerAddrs = strings.Split(addrs, ",")
	}
	cfg.UDPTrackerID = uint32(getEnvInt("UDP_TRACKER_ID", 1))
	cfg.UDPSyncIntervalMs = getEnvInt("UDP_SYNC_INTERVAL_MS", 10000)
	cfg.UDPDefaultShardRPS = uint64(getEnvInt64("UDP_DEFAULT_SHARD_RPS", 50_000))
	cfg.RegionCode = uint8(RegionCodeFromEnv())
	cfg.MultiRegionEnabled = getEnvBool("MULTI_REGION_ENABLED", false)
	cfg.NodeID = os.Getenv("NODE_ID")
	cfg.NodeRole = os.Getenv("NODE_ROLE")
	cfg.NodeScoreWindowMin = getEnvInt("NODE_SCORE_WINDOW_MIN", 15)
	cfg.NodeScoreMinSamples = getEnvInt("NODE_SCORE_MIN_SAMPLES", 30)
	cfg.NodeWarmupSec = getEnvInt("NODE_WARMUP_SEC", 300)
	cfg.ScoringWeightsJSON = os.Getenv("SCORING_WEIGHTS_JSON")
	cfg.OpLeaseTimeoutSec = getEnvInt("OP_LEASE_TIMEOUT_SEC", 30)
	cfg.OpLeaseMaxRenewals = getEnvInt("OP_LEASE_MAX_RENEWALS", 3)
	cfg.OpLeaseFencingDir = os.Getenv("OP_LEASE_FENCING_DIR")
	cfg.GlobalSpendBatchMin = getEnvInt("GLOBAL_SPEND_BATCH_MIN", 100)
	cfg.GlobalSpendFlushIntervalMs = getEnvInt("GLOBAL_SPEND_FLUSH_INTERVAL_MS", 500)
	cfg.GlobalSpendMaxConcurrency = getEnvInt("GLOBAL_SPEND_MAX_CONCURRENCY", 8)
	cfg.RegionProxyAddr = os.Getenv("REGION_PROXY_ADDR")
	if cfg.RegionProxyRedisURL == "" {
		cfg.RegionProxyRedisURL = os.Getenv("REGION_PROXY_REDIS_URL")
	}
	if cfg.RegionProxyRedisURL == "" && len(cfg.RedisAddrs) > 0 {
		cfg.RegionProxyRedisURL = "redis://" + cfg.RedisAddrs[0] + "/0"
	}
	loadDatabaseModules(cfg)
	cfg.ManagementURL = os.Getenv("MANAGEMENT_URL")
	if cfg.ManagementURL == "" && cfg.ManagementPort != "" {
		cfg.ManagementURL = "http://127.0.0.1:" + cfg.ManagementPort
	}

	loadControlplaneModules(cfg)

	cfg.IVT.Enabled = getEnvBool("IVT_DETECTOR_ENABLED", true)
	cfg.IVT.ScanIntervalMs = getEnvInt("IVT_DETECTOR_SCAN_INTERVAL_MS", 300000)
	cfg.IVT.OutboxPendingLimit = getEnvInt64("IVT_DETECTOR_OUTBOX_PENDING_LIMIT", 500)
	cfg.IVT.WindowSec = getEnvInt("IVT_DETECTOR_WINDOW_SEC", 3600)
	cfg.IVT.MinClicks = uint64(getEnvInt64("IVT_DETECTOR_MIN_CLICKS", 10))
	cfg.IVT.MinImpressions = uint64(getEnvInt64("IVT_DETECTOR_MIN_IMPRESSIONS", 1))
	cfg.IVT.ClickToImpRatio = getEnvFloat("IVT_DETECTOR_CLICK_TO_IMP_RATIO", 5.0)
	cfg.IVT.MinIPsPerUA = uint64(getEnvInt64("IVT_DETECTOR_MIN_IPS_PER_UA", 8))
	cfg.IVT.IntervalMinIntervals = uint64(getEnvInt64("IVT_DETECTOR_INTERVAL_MIN_INTERVALS", 30))
	cfg.IVT.IntervalMaxVariance = getEnvFloat("IVT_DETECTOR_INTERVAL_MAX_VARIANCE", 0.005)

	cfg.FraudScoring.Enabled = getEnvBool("FRAUD_SCORING_ENABLED", false)
	cfg.FraudScoring.ScanIntervalMs = getEnvInt("FRAUD_SCORING_SCAN_INTERVAL_MS", 300000)
	cfg.FraudScoring.BatchSize = getEnvInt("FRAUD_SCORING_BATCH_SIZE", 1000)
	cfg.FraudScoring.ModelPath = os.Getenv("FRAUD_SCORING_MODEL_PATH")
	if cfg.FraudScoring.ModelPath == "" {
		cfg.FraudScoring.ModelPath = "var/fraudscore/artifacts/model.txt"
	}
	cfg.FraudScoring.Standalone = getEnvBool("FRAUD_SCORER_STANDALONE", false)

	if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "" {
		cfg.AllowedOrigins = []string{"https://dashboard.example.com", "http://localhost:8188"}
	}

	cfg.Management.RetentionDays = getEnvInt("MANAGEMENT_RETENTION_DAYS", 90)
	cfg.Management.CancellationFeePercent = getEnvFloat("MANAGEMENT_CANCELLATION_FEE_PERCENT", 5.0)
	cfg.Management.ReconIntervalMs = getEnvInt("RECON_WORKER_INTERVAL_MS", 3_600_000)
	cfg.Management.ReconSnapshotIntervalMs = getEnvInt("RECON_SNAPSHOT_INTERVAL_MS", 0)
	cfg.Management.PacingIntervalMs = getEnvInt("PACING_CONTROLLER_INTERVAL_MS", 300_000)
	cfg.Management.RateLimitRPS = getEnvFloat("MANAGEMENT_RATE_LIMIT_RPS", 10)
	cfg.Management.RateLimitBurst = getEnvInt("MANAGEMENT_RATE_LIMIT_BURST", 50)
	cfg.Management.OpsAlertsEnabled = getEnvBool("OPS_ALERTS_ENABLED", false)
	cfg.Management.OpsAlertCooldownSec = getEnvInt("OPS_ALERT_COOLDOWN_SEC", 300)
	cfg.Management.DrainStuckThresholdSec = getEnvInt("OPS_ALERT_DRAIN_STUCK_SEC", 900)
	cfg.Management.BlacklistJanitorEnabled = getEnvBool("BLACKLIST_JANITOR_ENABLED", true)
	cfg.Management.BlacklistJanitorIntervalSec = getEnvInt("BLACKLIST_JANITOR_INTERVAL_SEC", 60)
	cfg.Management.BlacklistAutoTTLHours = getEnvInt("BLACKLIST_AUTO_TTL_HOURS", 24)
	cfg.Management.BlacklistFraudTTLHours = getEnvInt("BLACKLIST_FRAUD_TTL_HOURS", 168)
	cfg.Management.AlertmanagerWebhookEnabled = getEnvBool("ALERTMANAGER_WEBHOOK_ENABLED", false)
	cfg.Management.AlertmanagerWebhookToken = os.Getenv("ALERTMANAGER_WEBHOOK_TOKEN")
	cfg.Management.OpsAlertOutboxStuckSec = getEnvInt("OPS_ALERT_OUTBOX_STUCK_SEC", 120)
	cfg.Management.AuditExportPath = os.Getenv("AUDIT_EXPORT_PATH")
	if cfg.Management.AuditExportPath == "" {
		cfg.Management.AuditExportPath = "./data/audit-export"
	}
	cfg.Management.AuditExportRetentionDays = getEnvInt("AUDIT_EXPORT_RETENTION_DAYS", 90)
	cfg.Management.BillingExportPath = os.Getenv("BILLING_EXPORT_PATH")
	if cfg.Management.BillingExportPath == "" {
		cfg.Management.BillingExportPath = "./data/billing-export"
	}
	cfg.Management.SupplyExportPath = os.Getenv("SUPPLY_EXPORT_PATH")
	if cfg.Management.SupplyExportPath == "" {
		cfg.Management.SupplyExportPath = "./data/supply-export"
	}
	cfg.Management.AdminFanoutMaxConcurrency = getEnvInt("ADMIN_FANOUT_MAX_CONCURRENCY", 8)
	cfg.Management.LowBalanceThresholdMicro = int64(getEnvInt("LOW_BALANCE_THRESHOLD_MICRO", 5_000_000))
	cfg.Management.LowBalanceAlertEnabled = getEnvBool("LOW_BALANCE_ALERT_ENABLED", true)

	cfg.Control.EnableAuth = getEnvBool("CONTROL_ENABLE_AUTH", true)
	cfg.Control.EnableManagement = getEnvBool("CONTROL_ENABLE_MANAGEMENT", true)
	cfg.Control.EnablePayment = getEnvBool("CONTROL_ENABLE_PAYMENT", true)
	cfg.Control.EnableBilling = getEnvBool("CONTROL_ENABLE_BILLING", true)
	cfg.Control.EnableNotifier = getEnvBool("CONTROL_ENABLE_NOTIFIER", true)
	cfg.Control.EnableMarginGuard = getEnvBool("CONTROL_ENABLE_MARGIN_GUARD", true)
	cfg.Control.EnableCostSync = getEnvBool("CONTROL_ENABLE_COST_SYNC", true)

	cfg.GeoIP.DBPath = os.Getenv("GEOIP_DB_PATH")
	if cfg.GeoIP.DBPath == "" {
		cfg.GeoIP.DBPath = "deploy/geoip/GeoLite2-Country.mmdb"
	}
	cfg.GeoIP.StagingPath = os.Getenv("GEOIP_STAGING_PATH")
	if cfg.GeoIP.StagingPath == "" {
		cfg.GeoIP.StagingPath = cfg.GeoIP.DBPath + ".staging"
	}
	cfg.GeoIP.EditionID = os.Getenv("MAXMIND_EDITION_ID")
	if cfg.GeoIP.EditionID == "" {
		cfg.GeoIP.EditionID = "GeoLite2-Country"
	}
	cfg.GeoIP.LicenseKey = os.Getenv("MAXMIND_LICENSE_KEY")
	cfg.GeoIP.UpdaterEnabled = getEnvBool("GEOIP_UPDATER_ENABLED", false)
	cfg.GeoIP.UpdateIntervalHours = getEnvInt("GEOIP_UPDATE_INTERVAL_HOURS", 24)
	cfg.GeoIP.WatcherIntervalSec = getEnvInt("GEOIP_WATCHER_INTERVAL_SEC", 60)

	cfg.Lifecycle.ShutdownTimeoutMs = getEnvInt("SHUTDOWN_TIMEOUT_MS", 15000)
	cfg.Lifecycle.DrainTimeoutMs = getEnvInt("DRAIN_TIMEOUT_MS", 10000)
	cfg.Lifecycle.WaitTimeoutMs = getEnvInt("WAIT_TIMEOUT_MS", 5000)

	if cfg.ServerPort == "" {
		return nil, errors.New("SERVER_PORT is required")
	}
	if cfg.ProcessorPort == "" {
		cfg.ProcessorPort = "8186"
	}
	if cfg.ManagementPort == "" {
		cfg.ManagementPort = "8188"
	}
	if cfg.MetricsPort == "" {
		cfg.MetricsPort = "9090"
	}
	if cfg.DBDSN == "" {
		return nil, errors.New("DB_DSN is required")
	}
	if cfg.PaymentDBDSN == "" {
		cfg.PaymentDBDSN = cfg.DBDSN
	}
	if len(cfg.RedisAddrs) == 0 {
		return nil, errors.New("REDIS_ADDRS is required")
	}
	if cfg.Env == "production" && len(cfg.RedisAddrs) != ExpectedRedisShardCount {
		return nil, fmt.Errorf("production requires exactly %d Redis shards (REDIS_ADDRS), got %d", ExpectedRedisShardCount, len(cfg.RedisAddrs))
	}
	if cfg.RedisSentinelEnabled() {
		if len(cfg.RedisMasterNames) > 0 && len(cfg.RedisMasterNames) != len(cfg.RedisAddrs) {
			return nil, fmt.Errorf("REDIS_MASTER_NAMES count (%d) must match REDIS_ADDRS (%d)", len(cfg.RedisMasterNames), len(cfg.RedisAddrs))
		}
	}

	if cfg.RedisStreamName == "" {
		cfg.RedisStreamName = "ad:events:stream"
	}
	if cfg.FraudStreamName == "" {
		cfg.FraudStreamName = "ad:fraud:stream"
	}
	if cfg.RedisGroupName == "" {
		cfg.RedisGroupName = "ad:processor:group"
	}
	if cfg.RedisConsumerID == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown"
		}
		cfg.RedisConsumerID = hostname + ":" + strconv.Itoa(os.Getpid())
	}

	if cfg.AuthServerPort == "" {
		cfg.AuthServerPort = "51051"
	}
	if cfg.AuthMetricsPort == "" {
		cfg.AuthMetricsPort = "9091"
	}
	applyControlplaneDefaults(cfg)
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if cfg.TokenSymmetricKey == "" {
		return nil, errors.New("TOKEN_SYMMETRIC_KEY is required")
	}

	if cfg.FilterTimeoutMs <= 0 {
		cfg.FilterTimeoutMs = cfg.WriteTimeoutMs
	}
	if cfg.Env == "production" && cfg.FilterTimeoutMs > 100 {
		return nil, fmt.Errorf("production FILTER_TIMEOUT_MS must be <= 100 (got %d)", cfg.FilterTimeoutMs)
	}
	if cfg.Env == "production" && cfg.TrackerPGFallback {
		return nil, fmt.Errorf("production TRACKER_PG_FALLBACK must be 0")
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

func (c *Config) NotifierDialEnabled() bool {
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
