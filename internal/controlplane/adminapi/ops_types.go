package adminapi

import (
	"context"
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
	ExportAuditCSV(ctx context.Context, cursor string, w io.Writer) (AuditExportResult, error)
	LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error)
	ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error)
	GetDashboardSummary(ctx context.Context) (DashboardSummaryDTO, error)
	GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (DashboardMetricsDTO, error)
	GetMLModelStatus(ctx context.Context) (MLModelStatusDTO, error)
	AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error
	ListMLManualLabels(ctx context.Context) ([]MLManualLabelDTO, error)
}
