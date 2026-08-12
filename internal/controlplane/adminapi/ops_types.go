package adminapi

import (
	"context"
	"encoding/json"
	"io"
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

type OutboxHealthSummary struct {
	Pending              int64   `json:"pending"`
	OldestPendingSeconds float64 `json:"oldest_pending_seconds"`
	LastProcessedEventID int64   `json:"last_processed_event_id"`
}

type ShardHealthStatus struct {
	ShardID             int     `json:"shard_id"`
	PingOK              bool    `json:"ping_ok"`
	PingError           string  `json:"ping_error,omitempty"`
	PingLatencyMs       float64 `json:"ping_latency_ms,omitempty"`
	ConfigVersion       *int64  `json:"config_version,omitempty"`
	ConfigVersionLag    int64   `json:"config_version_lag"`
	ConfigVersionSynced bool    `json:"config_version_synced"`
}

type ShardHealthReport struct {
	EmergencyBreaker string              `json:"emergency_breaker"`
	Outbox           OutboxHealthSummary `json:"outbox"`
	Shards           []ShardHealthStatus `json:"shards"`
}

type IncidentSnapshotDTO struct {
	EmergencyBreaker string              `json:"emergency_breaker"`
	Shards           []ShardHealthStatus `json:"shards"`
	Outbox           OutboxHealthSummary `json:"outbox"`
	StreamLag        []ShardStreamLag    `json:"stream_lag"`
	BreakerStates    map[string]string   `json:"breaker_states"`
	Partial          bool                `json:"partial"`
	Errors           []FanOutSourceError `json:"errors,omitempty"`
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
	EnqueueDLQRetry(ctx context.Context, payload DLQRetryPayload, idempotencyKey string) error
	GetShardHealthFanOut(ctx context.Context) (ShardHealthAPIResponse, error)
	ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (AuditExportResult, error)
	LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error)
	ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error)
	GetDashboardSummary(ctx context.Context) (DashboardSummaryDTO, error)
	GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (DashboardMetricsDTO, error)
	GetMLModelStatus(ctx context.Context) (MLModelStatusDTO, error)
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
	GeneratedAt      string                 `json:"generated_at"`
	Services         []DashboardServiceCard `json:"services"`
	DriftMicroMax    float64                `json:"drift_micro_max"`
	DriftAlert       bool                   `json:"drift_alert"`
	RPSEstimate      float64                `json:"rps_estimate"`
	OutboxPending    int64                  `json:"outbox_pending"`
	EmergencyBreaker string                 `json:"emergency_breaker"`
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

type MLManualLabelRequest struct {
	IPHash string `json:"ip_hash"`
	Label  int    `json:"label"`
	Reason string `json:"reason"`
}

type MLManualLabelDTO struct {
	IPHash    string `json:"ip_hash"`
	Label     int    `json:"label"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

type SupportBundleWriter interface {
	WriteSupportBundle(ctx context.Context, w io.Writer) error
}
