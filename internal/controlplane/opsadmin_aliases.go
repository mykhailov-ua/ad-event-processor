package controlplane

import (
	"context"

	"ad-event-processor/internal/opsadmin"
)

type (
	OpsHTTPHandlers          = opsadmin.HTTPHandlers
	ManagementOpsReader      = opsadmin.ManagementOpsReader
	IncidentSnapshotDTO      = opsadmin.IncidentSnapshotDTO
	OutboxEventDTO           = opsadmin.OutboxEventDTO
	OutboxListResult         = opsadmin.OutboxListResult
	DLQEntryDTO              = opsadmin.DLQEntryDTO
	DLQInboxEntryDTO         = opsadmin.DLQInboxEntryDTO
	DLQInboxListResult       = opsadmin.DLQInboxListResult
	DLQRetryPayload          = opsadmin.DLQRetryPayload
	ConsentProofDTO          = opsadmin.ConsentProofDTO
	ConsentProofListResult   = opsadmin.ConsentProofListResult
	ConsentRecord            = opsadmin.ConsentRecord
	ConsentRecorder          = opsadmin.ConsentRecorder
	ConsentVerifier          = opsadmin.ConsentVerifier
	DomainRotationHostDTO    = opsadmin.DomainRotationHostDTO
	DomainRotationListResult = opsadmin.DomainRotationListResult
	ShardHealthAPIResponse   = opsadmin.ShardHealthAPIResponse
	ShardHealthReport        = opsadmin.ShardHealthReport
	ShardHealthStatus        = opsadmin.ShardHealthStatus
	ShardStreamLag           = opsadmin.ShardStreamLag
	OutboxHealthSummary      = opsadmin.OutboxHealthSummary
	AuditExportResult        = opsadmin.AuditExportResult
	ReconRunDTO              = opsadmin.ReconRunDTO
	AuditLister              = opsadmin.AuditLister
	RolesReloader            = opsadmin.RolesReloader
	FraudThreatEnqueuer      = opsadmin.FraudThreatEnqueuer
	FraudThreatEnqueueItem   = opsadmin.FraudThreatEnqueueItem
	BlacklistAdmin           = opsadmin.BlacklistAdmin
	Shard0CatchupRunner      = opsadmin.Shard0CatchupRunner
	DashboardServiceCard     = opsadmin.DashboardServiceCard
	DashboardSummaryDTO      = opsadmin.DashboardSummaryDTO
	DashboardMetricPoint     = opsadmin.DashboardMetricPoint
	DashboardMetricsDTO      = opsadmin.DashboardMetricsDTO
	MLModelVersionDTO        = opsadmin.MLModelVersionDTO
	MLModelRedisDTO          = opsadmin.MLModelRedisDTO
	MLShardSyncDTO           = opsadmin.MLShardSyncDTO
	MLFeatureImportanceDTO   = opsadmin.MLFeatureImportanceDTO
	MLModelStatusDTO         = opsadmin.MLModelStatusDTO
	MLEvalMetricsBlockDTO    = opsadmin.MLEvalMetricsBlockDTO
	MLEvalReportDTO          = opsadmin.MLEvalReportDTO
	SupportBundleWriter      = opsadmin.SupportBundleWriter
	StackHealthSnapshot      = opsadmin.StackHealthSnapshot
	ClientRUMIngestDTO       = opsadmin.ClientRUMIngestDTO
	RUMStore                 = opsadmin.RUMStore
	AffectedCampaignDTO      = opsadmin.AffectedCampaignDTO
	FanOutSourceError        = opsadmin.FanOutSourceError
	FanOutCollector          = opsadmin.FanOutCollector
)

type FanOutResult[T any] = opsadmin.FanOutResult[T]

type FanOutSource[T any] = opsadmin.FanOutSource[T]

var NewFanOutCollector = opsadmin.NewFanOutCollector

var (
	DecodeFanOutCursor = opsadmin.DecodeFanOutCursor
	EncodeFanOutCursor = opsadmin.EncodeFanOutCursor
)

var ErrDLQEntryNotFound = opsadmin.ErrDLQEntryNotFound

var DefaultEmptyAuditedMetrics = opsadmin.DefaultEmptyAuditedMetrics

var (
	dlqRouteID                 = opsadmin.DLQRouteID
	dlqInboxSourceFromProvider = opsadmin.DLQInboxSourceFromProvider
	parseInboxStreamShard      = opsadmin.ParseInboxStreamShard
	parseInboxStreamEntryID    = opsadmin.ParseInboxStreamEntryID
	parseDLQShardFromRoute     = opsadmin.ParseDLQShardFromRoute
	parseDLQEntryIDFromRoute   = opsadmin.ParseDLQEntryIDFromRoute
	parseMLEvalReportJSON      = opsadmin.ParseMLEvalReportJSON
	normalizeMLEvalReport      = opsadmin.NormalizeMLEvalReport
	topFeatureImportance       = opsadmin.TopFeatureImportance
)

func CollectFanOut[T any](ctx context.Context, c *FanOutCollector, sources []FanOutSource[T]) FanOutResult[T] {
	return opsadmin.CollectFanOut(ctx, c, sources)
}
