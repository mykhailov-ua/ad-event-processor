package opsadmin

import (
	"context"
	"encoding/json"
	"io"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/shardadmin"
)

type DLQRetryPayload struct {
	ShardID int    `json:"shard_id"`
	Stream  string `json:"stream"`
	EntryID string `json:"entry_id"`
	DLQID   string `json:"dlq_id"`
}

type OutboxEventDTO struct {
	ID        int64  `json:"id"`
	EventType string `json:"event_type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type OutboxListResult struct {
	Items      []OutboxEventDTO `json:"items"`
	Total      int64            `json:"total"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type DLQEntryDTO struct {
	ID         string `json:"id"`
	ShardID    int    `json:"shard_id"`
	StreamID   string `json:"stream_id"`
	EntryID    string `json:"entry_id"`
	CampaignID string `json:"campaign_id,omitempty"`
	EventType  string `json:"event_type,omitempty"`
	Error      string `json:"error,omitempty"`
	FailedAt   string `json:"failed_at"`
	RetryCount int32  `json:"retry_count"`
	WorkerID   string `json:"worker_id,omitempty"`
}

type ShardStreamLag struct {
	ShardID   int    `json:"shard_id"`
	Stream    string `json:"stream"`
	Length    int64  `json:"length"`
	DLQLength int64  `json:"dlq_length"`
}

type OutboxHealthSummary = shardadmin.OutboxHealthSummary

type ShardHealthStatus = shardadmin.ShardHealthStatus

type ShardHealthReport = shardadmin.ShardHealthReport

type IncidentSnapshotDTO struct {
	EmergencyBreaker  string                `json:"emergency_breaker"`
	Shards            []ShardHealthStatus   `json:"shards"`
	Outbox            OutboxHealthSummary   `json:"outbox"`
	StreamLag         []ShardStreamLag      `json:"stream_lag"`
	BreakerStates     map[string]string     `json:"breaker_states"`
	Partial           bool                  `json:"partial"`
	Errors            []FanOutSourceError   `json:"errors,omitempty"`
	StaleDashboard    bool                  `json:"stale_dashboard,omitempty"`
	AffectedCampaigns []AffectedCampaignDTO `json:"affected_campaigns,omitempty"`
}

type AffectedCampaignDTO struct {
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name,omitempty"`
}

type DLQInboxEntryDTO struct {
	ID              string `json:"id"`
	Source          string `json:"source"`
	CampaignID      string `json:"campaign_id,omitempty"`
	EventType       string `json:"event_type,omitempty"`
	Error           string `json:"error,omitempty"`
	FailedAt        string `json:"failed_at,omitempty"`
	FailedAtDisplay string `json:"failed_at_display,omitempty"`
	Status          string `json:"status,omitempty"`
	RetryCount      int32  `json:"retry_count,omitempty"`
	ShardID         int    `json:"shard_id,omitempty"`
	StreamID        string `json:"stream_id,omitempty"`
	EntryID         string `json:"entry_id,omitempty"`
	ClickID         string `json:"click_id,omitempty"`
	Provider        string `json:"provider,omitempty"`
}

type DLQInboxListResult struct {
	Items      []DLQInboxEntryDTO  `json:"items"`
	NextCursor string              `json:"next_cursor,omitempty"`
	Partial    bool                `json:"partial,omitempty"`
	Errors     []FanOutSourceError `json:"errors,omitempty"`
}

type ConsentProofDTO struct {
	ID         int64  `json:"id"`
	UserIDHash string `json:"user_id_hash"`
	Purposes   int16  `json:"purposes"`
	Source     string `json:"source"`
	RecordedAt string `json:"recorded_at"`
	AdStorage  bool   `json:"ad_storage"`
	Analytics  bool   `json:"analytics_storage"`
}

type ConsentProofListResult struct {
	Items      []ConsentProofDTO `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type DomainRotationHostDTO struct {
	Hostname            string `json:"hostname"`
	Role                string `json:"role"`
	HealthStatus        string `json:"health_status"`
	SSLStatus           string `json:"ssl_status,omitempty"`
	PoolID              string `json:"pool_id,omitempty"`
	PoolDomainStatus    string `json:"pool_domain_status,omitempty"`
	DmrCampaignCount    int64  `json:"dmr_campaign_count"`
	ActiveCampaignCount int64  `json:"active_campaign_count"`
}

type DomainRotationListResult struct {
	Hosts []DomainRotationHostDTO `json:"hosts"`
}

type ShardHealthAPIResponse struct {
	ShardHealthReport
	Partial bool                `json:"partial"`
	Errors  []FanOutSourceError `json:"errors,omitempty"`
}

type AuditExportResult struct {
	NextCursor string
	Truncated  bool
	Bytes      int
}

type ReconRunDTO struct {
	Service            string `json:"service"`
	ID                 int64  `json:"id"`
	PeriodStart        string `json:"period_start"`
	PeriodEnd          string `json:"period_end"`
	Status             string `json:"status"`
	TotalDelta         *int64 `json:"total_delta,omitempty"`
	CampaignsChecked   *int32 `json:"campaigns_checked,omitempty"`
	DiscrepanciesFound *int32 `json:"discrepancies_found,omitempty"`
	FindingsCount      *int32 `json:"findings_count,omitempty"`
	IntentsChecked     *int32 `json:"intents_checked,omitempty"`
	ErrorMessage       string `json:"error_message,omitempty"`
	CreatedAt          string `json:"created_at"`
	CompletedAt        string `json:"completed_at,omitempty"`
}

type ManagementOpsReader interface {
	GetIncidentSnapshot(ctx context.Context) (IncidentSnapshotDTO, error)
	ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (OutboxListResult, error)
	ListDLQEntries(ctx context.Context, cursor string, limit int) (FanOutResult[DLQEntryDTO], error)
	ListDLQInbox(ctx context.Context, source, cursor string, limit int) (DLQInboxListResult, error)
	RetryDLQInbox(ctx context.Context, source, id, idempotencyKey string) error
	EnqueueDLQRetry(ctx context.Context, payload DLQRetryPayload, idempotencyKey string) error
	ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (ConsentProofListResult, error)
	ListDomainRotation(ctx context.Context) (DomainRotationListResult, error)
	GetShardHealthFanOut(ctx context.Context) (ShardHealthAPIResponse, error)
	ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (AuditExportResult, error)
	LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error)
	ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error)
	GetDashboardSummary(ctx context.Context) (DashboardSummaryDTO, error)
	GetStackHealthSnapshot(ctx context.Context) (StackHealthSnapshot, error)
	GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (DashboardMetricsDTO, error)
	GetMLModelStatus(ctx context.Context) (MLModelStatusDTO, error)
	GetMLEvalReport(ctx context.Context) (MLEvalReportDTO, error)
	AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error
	ListMLManualLabels(ctx context.Context) ([]MLManualLabelDTO, error)
}

type AuditLister interface {
	ListAuditLogs(ctx context.Context, limit, offset int32, redactPII bool) ([]AuditLogDTO, int64, error)
}

type ConsentRecord struct {
	UserID    string `json:"user_id"`
	Purposes  int16  `json:"purposes"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp,omitempty"`
}

type ConsentRecorder interface {
	RecordConsent(ctx context.Context, in ConsentRecord) error
}

type ConsentVerifier interface {
	Verify(body []byte, signature string) error
}

type RolesReloader interface {
	ReloadRoles() error
	RolesPath() string
}

type FraudThreatEnqueuer interface {
	EnqueueFraudThreat(ctx context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error
	EnqueueFraudThreatBatch(ctx context.Context, items []FraudThreatEnqueueItem) (int, error)
}

type FraudThreatEnqueueItem struct {
	Action     string  `json:"action"`
	IP         string  `json:"ip"`
	CampaignID string  `json:"campaign_id"`
	Score      float64 `json:"score"`
	Boost      int32   `json:"boost"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

type BlacklistAdmin interface {
	BlockIPWithTTL(ctx context.Context, ip, source string, ttlSeconds *int64) error
	PreviewBlockIP(ctx context.Context, ip, source string, ttlSeconds *int64) (MutationPreviewDTO, error)
	UnblockIP(ctx context.Context, ip, source string) error
	ListBlacklist(ctx context.Context, limit, offset int32) ([]BlacklistDTO, int64, error)
}

type Shard0CatchupRunner interface {
	RunShard0Catchup(ctx context.Context) error
}

type shard0CatchupResponse struct {
	Status string `json:"status"`
}

type DashboardServiceCard struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type DashboardSummaryDTO struct {
	GeneratedAt        string                   `json:"generated_at"`
	GeneratedAtDisplay string                   `json:"generated_at_display,omitempty"`
	Services           []DashboardServiceCard   `json:"services"`
	DriftMicroMax      float64                  `json:"drift_micro_max"`
	DriftAlert         bool                     `json:"drift_alert"`
	RPSEstimate        float64                  `json:"rps_estimate"`
	OutboxPending      int64                    `json:"outbox_pending"`
	EmergencyBreaker   string                   `json:"emergency_breaker"`
	Infra              InfraResourceSnapshotDTO `json:"infra,omitempty"`
}

type InfraResourceSnapshotDTO struct {
	HeapInuseBytes    int64   `json:"heap_inuse_bytes,omitempty"`
	RSSBytes          int64   `json:"rss_bytes,omitempty"`
	Goroutines        int64   `json:"goroutines,omitempty"`
	GnetConnections   int64   `json:"gnet_active_connections,omitempty"`
	CPUUtilizationPct float64 `json:"cpu_utilization_pct,omitempty"`
	HTTPRPS           float64 `json:"http_rps,omitempty"`
	ScrapeStale       bool    `json:"scrape_stale,omitempty"`
	ObservedAt        string  `json:"observed_at,omitempty"`
}

type DashboardMetricPoint struct {
	Name       string  `json:"name"`
	LabelsHash string  `json:"labels_hash,omitempty"`
	Timestamp  string  `json:"ts"`
	Value      float64 `json:"value"`
}

type DashboardMetricsDTO struct {
	Range       string                 `json:"range"`
	BucketSec   int                    `json:"bucket_sec"`
	Points      []DashboardMetricPoint `json:"points"`
	GeneratedAt string                 `json:"generated_at"`
}

type MLModelVersionDTO struct {
	ID               string          `json:"id"`
	ArtifactHash     string          `json:"artifact_hash"`
	Status           string          `json:"status"`
	CreatedAt        string          `json:"created_at"`
	ArtifactMetadata json.RawMessage `json:"artifact_metadata,omitempty"`
}

type MLModelRedisDTO struct {
	VersionID        string `json:"version_id,omitempty"`
	Hash             string `json:"hash,omitempty"`
	AppliedAt        string `json:"applied_at,omitempty"`
	ShardsReporting  int    `json:"shards_reporting"`
	ShardsConsistent bool   `json:"shards_consistent"`
}

type MLShardSyncDTO struct {
	ShardID      int    `json:"shard_id"`
	ModelVersion string `json:"model_version"`
	Phase        string `json:"phase"`
	StartedAt    string `json:"started_at"`
}

type MLFeatureImportanceDTO struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type MLModelStatusDTO struct {
	ActiveVersion  *MLModelVersionDTO       `json:"active_version,omitempty"`
	SyncingVersion *MLModelVersionDTO       `json:"syncing_version,omitempty"`
	Redis          MLModelRedisDTO          `json:"redis"`
	ShardSync      []MLShardSyncDTO         `json:"shard_sync"`
	Drift          json.RawMessage          `json:"drift,omitempty"`
	DriftDetected  bool                     `json:"drift_detected"`
	Precision      float64                  `json:"precision,omitempty"`
	Recall         float64                  `json:"recall,omitempty"`
	Importance     []MLFeatureImportanceDTO `json:"importance,omitempty"`
}

type MLEvalMetricsBlockDTO struct {
	Status            string  `json:"status"`
	LabelMethod       string  `json:"label_method,omitempty"`
	LabelDefinition   string  `json:"label_definition,omitempty"`
	LabeledRows       int64   `json:"labeled_rows"`
	MatchedRows       int64   `json:"matched_rows,omitempty"`
	Confidence        string  `json:"confidence,omitempty"`
	Precision         float64 `json:"precision,omitempty"`
	Recall            float64 `json:"recall,omitempty"`
	F1                float64 `json:"f1,omitempty"`
	FalsePositiveRate float64 `json:"false_positive_rate,omitempty"`
	TP                int64   `json:"tp,omitempty"`
	FP                int64   `json:"fp,omitempty"`
	FN                int64   `json:"fn,omitempty"`
	TN                int64   `json:"tn,omitempty"`
}

type MLEvalReportDTO struct {
	Status         string                `json:"status"`
	GeneratedAt    string                `json:"generated_at,omitempty"`
	Hours          int                   `json:"hours,omitempty"`
	Threshold      float64               `json:"threshold,omitempty"`
	ProxyMetrics   MLEvalMetricsBlockDTO `json:"proxy_metrics"`
	AuditedMetrics MLEvalMetricsBlockDTO `json:"audited_metrics"`
	Drift          json.RawMessage       `json:"drift,omitempty"`
	DriftDetected  bool                  `json:"drift_detected,omitempty"`
}

type SupportBundleWriter interface {
	WriteSupportBundle(ctx context.Context, w io.Writer) error
}

type StackHealthSnapshot struct {
	Status                          string   `json:"status"`
	ClickHouseLagSeconds            float64  `json:"clickhouse_lag_seconds"`
	OutboxOldestPendingSeconds      float64  `json:"outbox_oldest_pending_seconds"`
	RedisShardReachable             bool     `json:"redis_shard_reachable"`
	RedisShardsReachable            int      `json:"redis_shards_reachable"`
	RedisShardsTotal                int      `json:"redis_shards_total"`
	CostSyncLastSuccessSeconds      *float64 `json:"cost_sync_last_success_seconds,omitempty"`
	AutomationWorkerLastTickSeconds *float64 `json:"automation_worker_last_tick_seconds,omitempty"`
	LicenseState                    string   `json:"license_state"`
}

type OffsetListResponse[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type PaymentIntentListResponse = OffsetListResponse[PaymentHistoryRow]

type BlacklistListResponse struct {
	Items []BlacklistDTO `json:"items"`
	Total int64          `json:"total"`
}

type (
	AuditLogDTO                   = campaign.AuditLogDTO
	MutationPreviewDTO            = campaign.MutationPreviewDTO
	BlacklistDTO                  = campaign.BlacklistDTO
	MLManualLabelDTO              = fraudadmin.MLManualLabelDTO
	MLManualLabelRequest          = fraudadmin.FraudManualLabelRequest
	FraudPresetsService           = fraudadmin.PresetsService
	PatchFraudPolicyPresetRequest = fraudadmin.PatchFraudPolicyPresetRequest
)
