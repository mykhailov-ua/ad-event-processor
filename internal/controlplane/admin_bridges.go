package controlplane

import (
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/fraud"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/piihash"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type fraudLabelsAPIAdapter struct {
	svc *Service
}

type fraudLabelsHost struct {
	svc *Service
}

func (h fraudLabelsHost) LabelsPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (s *Service) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]fraudadmin.MLManualLabelDTO, error) {
	return fraudadmin.NewLabels(fraudLabelsHost{svc: s}).ListMLManualLabelsForCustomer(ctx, customerID, limit)
}

func (a fraudLabelsAPIAdapter) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]fraudadmin.MLManualLabelDTO, error) {
	return fraudadmin.NewLabels(fraudLabelsHost{svc: a.svc}).ListMLManualLabelsForCustomer(ctx, customerID, limit)
}

func (a fraudLabelsAPIAdapter) UpsertMLManualLabelForCustomer(ctx context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error {
	return fraudadmin.NewLabels(fraudLabelsHost{svc: a.svc}).UpsertMLManualLabelForCustomer(ctx, customerID, ipHash, label, reason)
}

func (a fraudLabelsAPIAdapter) BulkUpsertMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, rows []fraudadmin.FraudManualLabelRow) (int, error) {
	return fraudadmin.NewLabels(fraudLabelsHost{svc: a.svc}).BulkUpsertMLManualLabelsForCustomer(ctx, customerID, rows)
}

type fraudDecisionsAPIAdapter struct {
	svc *Service
}

func (a fraudDecisionsAPIAdapter) ExplainFraudDecision(ctx context.Context, customerID uuid.UUID, ipHash string, campaignID *uuid.UUID, hours int) (fraudadmin.FraudDecisionDTO, error) {
	return fraudadmin.ExplainFraudDecision(ctx, fraudDecisionsHost{svc: a.svc}, customerID, ipHash, campaignID, hours)
}

func mapFraudadminErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, fraudadmin.ErrValidation) {
		return errValidation(err.Error())
	}
	if errors.Is(err, fraudadmin.ErrFraudDecisionNotFound) {
		return ErrFraudDecisionNotFound
	}
	return err
}

type fraudIntegrationsAPIAdapter struct {
	svc *Service
}

func (a fraudIntegrationsAPIAdapter) ListFraudIntegrationsForCustomer(ctx context.Context, customerID uuid.UUID) ([]fraudadmin.FraudIntegrationDTO, error) {
	rows, err := fraudadmin.ListIntegrationsForCustomer(ctx, a.svc.GetPool(), customerID)
	if err != nil {
		return nil, mapFraudadminErr(err)
	}
	return rows, nil
}

type fraudOverridesAPIAdapter struct {
	svc *Service
}

func (a fraudOverridesAPIAdapter) ApplyFraudScoringOverrideForCustomer(ctx context.Context, customerID uuid.UUID, req fraudadmin.FraudOverrideRequest) error {
	return mapFraudadminErr(fraudadmin.ApplyFraudScoringOverrideForCustomer(ctx, fraudOverridesHost{svc: a.svc}, customerID, req))
}

type fraudPresetsAPIAdapter struct {
	svc *Service
}

func (a fraudPresetsAPIAdapter) ListFraudPolicyPresets(ctx context.Context) ([]fraudadmin.FraudPolicyPresetDTO, error) {
	return fraudadmin.ListPolicyPresets(ctx, a.svc.GetPool())
}

func (a fraudPresetsAPIAdapter) UpdateFraudPolicyPreset(ctx context.Context, name string, req fraudadmin.PatchFraudPolicyPresetRequest) (fraudadmin.FraudPolicyPresetDTO, error) {
	out, err := fraudadmin.UpdatePolicyPreset(ctx, fraudPresetsHost{svc: a.svc}, name, req)
	if err != nil {
		return fraudadmin.FraudPolicyPresetDTO{}, mapFraudadminErr(err)
	}
	return out, nil
}

type fraudPresetsHost struct {
	svc *Service
}

func (h fraudPresetsHost) PresetsPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h fraudPresetsHost) PresetActorID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (h fraudPresetsHost) PresetAuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, name string, pass, suspect, ivt, block uint8) {
	h.svc.AuditLog(ctx, q, adminID, "UPDATE_FRAUD_POLICY_PRESET", "system", nil, map[string]any{
		"name":    name,
		"pass":    pass,
		"suspect": suspect,
		"ivt":     ivt,
		"block":   block,
	}, nil)
}

type campaignFraudAPIAdapter struct {
	svc *Service
}

func (a campaignFraudAPIAdapter) GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (CampaignFraudConfigDTO, error) {
	return fraudadmin.GetCampaignFraudConfig(ctx, fraudCampaignConfigHost{svc: a.svc}, campaignID)
}

func (a campaignFraudAPIAdapter) UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, req PatchCampaignFraudRequest) (CampaignFraudConfigDTO, error) {
	out, err := fraudadmin.UpdateCampaignFraudConfig(ctx, fraudCampaignConfigHost{svc: a.svc}, campaignID, req)
	return out, mapFraudadminErr(err)
}

func (a campaignFraudAPIAdapter) PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error) {
	out, err := fraudadmin.PreviewCampaignFraudImpact(ctx, fraudCampaignConfigHost{svc: a.svc}, campaignID, req)
	return out, mapFraudadminErr(err)
}

var _ fraudadmin.BlacklistJanitorHost = (*Service)(nil)

func (s *Service) BlacklistJanitorAlerter() fraudadmin.BlacklistJanitorAlerter {
	if s == nil {
		return nil
	}
	return s.alerter
}

type fraudDecisionsHost struct {
	svc *Service
}

func (h fraudDecisionsHost) DecisionsPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h fraudDecisionsHost) DecisionsClickHouse() *database.ClickHouseQuery {
	if h.svc == nil {
		return nil
	}
	return h.svc.clickhouseQuery
}

func (h fraudDecisionsHost) FraudExplainLiveScoreEnabled() bool {
	return h.svc != nil && h.svc.cfg != nil && h.svc.cfg.FraudScoring.ExplainLiveScore
}

func (h fraudDecisionsHost) FraudExplainScorer(ctx context.Context) (fraud.Scorer, error) {
	return h.svc.fraudExplainScorer()
}

type fraudOverridesHost struct {
	svc *Service
}

func (h fraudOverridesHost) OverridesPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h fraudOverridesHost) OverrideActorID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (h fraudOverridesHost) OverrideAuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any) {
	h.svc.AuditLog(ctx, q, adminID, action, targetType, targetID, changes, metadata)
}

func (h fraudOverridesHost) OverrideHashIP(ip string) ([16]byte, error) {
	if h.svc == nil || h.svc.cfg == nil {
		return [16]byte{}, fmt.Errorf("piihash: service not configured")
	}
	hasher, err := piihash.NewFromSalt(h.svc.cfg.PIISaltVersion, string(h.svc.cfg.PIISaltHex), string(h.svc.cfg.TokenSymmetricKey))
	if err != nil {
		return [16]byte{}, fmt.Errorf("piihash: %w", err)
	}
	return hasher.HashIP(ip), nil
}

func (h fraudOverridesHost) OverrideEnqueueClearBoost(ctx context.Context, q db.Querier, campaignID string) error {
	payload, err := coldpath.MarshalOutbox(outbox.FraudThreatPayload{
		Action:     "boost",
		CampaignID: campaignID,
		Boost:      0,
		TTLSeconds: 0,
	})
	if err != nil {
		return err
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "ML_SCORE_BOOST",
		Payload:   payload,
	})
	return err
}

func (h fraudOverridesHost) OverrideEnqueueBlacklistRemove(ctx context.Context, q db.Querier, ip string) error {
	payload, err := coldpath.MarshalOutbox(outbox.BlacklistPayload{Action: "remove", IP: ip, Reason: "fraud"})
	if err != nil {
		return fmt.Errorf("marshal blacklist outbox payload: %w", err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "UPDATE_BLACKLIST",
		Payload:   payload,
	})
	return err
}

type fraudCampaignConfigHost struct {
	svc *Service
}

func (h fraudCampaignConfigHost) ConfigPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h fraudCampaignConfigHost) ConfigClickHouse() *database.ClickHouseQuery {
	if h.svc == nil {
		return nil
	}
	return h.svc.clickhouseQuery
}

func (h fraudCampaignConfigHost) ConfigActorID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

func (h fraudCampaignConfigHost) ConfigAuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, campaignID uuid.UUID, changes fraudadmin.CampaignFraudAuditChange) {
	h.svc.AuditLog(ctx, q, adminID, "UPDATE_CAMPAIGN_FRAUD", "campaign", &campaignID, platformadmin.AuditCampaignFraudChange{
		FraudThresholdPass:       changes.FraudThresholdPass,
		FraudThresholdSuspect:    changes.FraudThresholdSuspect,
		FraudThresholdIVT:        changes.FraudThresholdIVT,
		FraudThresholdBlock:      changes.FraudThresholdBlock,
		SilentRejectEnabled:      changes.SilentRejectEnabled,
		BehaviorFlags:            changes.BehaviorFlags,
		CanvasRetestEnabled:      changes.CanvasRetestEnabled,
		CgnatIPPolicyEnabled:     changes.CgnatIPPolicyEnabled,
		AcceptLangGeoEnabled:     changes.AcceptLangGeoEnabled,
		JSONSerializationEnabled: changes.JSONSerializationEnabled,
	}, nil)
}

func (h fraudCampaignConfigHost) ConfigResolvePresetThresholds(ctx context.Context, name string) (uint8, uint8, uint8, uint8, error) {
	pass, suspect, ivt, block, err := fraudadmin.ResolvePresetThresholds(ctx, h.svc.GetPool(), name)
	if err != nil {
		return 0, 0, 0, 0, mapFraudadminErr(err)
	}
	return pass, suspect, ivt, block, nil
}

func (h fraudCampaignConfigHost) ConfigEnqueueUpdateCampaignFraud(ctx context.Context, q db.Querier, campaignID uuid.UUID) error {
	payload, err := coldpath.MarshalOutbox(outbox.CampaignIDPayload{CampaignID: campaignID.String()})
	if err != nil {
		return fmt.Errorf("marshal update campaign fraud outbox payload: %w", err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "UPDATE_CAMPAIGN_FRAUD",
		Payload:   payload,
	})
	return err
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

func (s *Service) GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (CampaignFraudConfigDTO, error) {
	out, err := fraudadmin.GetCampaignFraudConfig(ctx, fraudCampaignConfigHost{svc: s}, campaignID)
	return out, mapFraudadminErr(err)
}

func (s *Service) PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error) {
	out, err := fraudadmin.PreviewCampaignFraudImpact(ctx, fraudCampaignConfigHost{svc: s}, campaignID, req)
	return out, mapFraudadminErr(err)
}

func (s *Service) UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, upd PatchCampaignFraudRequest) (CampaignFraudConfigDTO, error) {
	out, err := fraudadmin.UpdateCampaignFraudConfig(ctx, fraudCampaignConfigHost{svc: s}, campaignID, upd)
	return out, mapFraudadminErr(err)
}

func (s *Service) ApplyFraudScoringOverride(ctx context.Context, req FraudScoringOverrideRequest) error {
	return mapFraudadminErr(fraudadmin.ApplyFraudScoringOverride(ctx, fraudOverridesHost{svc: s}, req))
}

func (s *Service) CheckAndHandleStaleEpochs(ctx context.Context) error {
	return fraudadmin.CheckAndHandleStaleEpochs(ctx, fraudStaleEpochsHost{svc: s})
}

type fraudMLSnapshotHost struct {
	svc *Service
}

func (h fraudMLSnapshotHost) SnapshotPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h fraudMLSnapshotHost) SnapshotRedisShards() []redis.UniversalClient {
	if h.svc == nil {
		return nil
	}
	return h.svc.redisShards
}

type fraudMLShadowDeltaHost struct {
	svc *Service
}

func (h fraudMLShadowDeltaHost) SnapshotPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h fraudMLShadowDeltaHost) ClickHouseQuery() *database.ClickHouseQuery {
	if h.svc == nil {
		return nil
	}
	return h.svc.clickhouseQuery
}

func (s *Service) StartMLShadowDeltaSnapshotWorker(ctx context.Context) {
	if s == nil {
		return
	}
	worker := fraudadmin.NewMLShadowDeltaSnapshotWorker(fraudMLShadowDeltaHost{svc: s})
	s.StartBackgroundWorker(func() {
		worker.Start(ctx)
	})
	slog.Info("ml shadow delta snapshot worker starting")
}

type fraudStaleEpochsHost struct {
	svc *Service
}

func (h fraudStaleEpochsHost) StaleEpochsRedisShards() []redis.UniversalClient {
	if h.svc == nil {
		return nil
	}
	return h.svc.redisShards
}

func (h fraudStaleEpochsHost) StaleEpochsPool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h fraudStaleEpochsHost) StaleEpochsUpdateSettings(ctx context.Context, settings map[string]string) error {
	return h.svc.UpdateSettings(ctx, settings)
}

var (
	_ fraudadmin.DecisionsHost             = fraudDecisionsHost{}
	_ fraudadmin.OverridesHost             = fraudOverridesHost{}
	_ fraudadmin.CampaignConfigHost        = fraudCampaignConfigHost{}
	_ fraudadmin.StaleEpochsHost           = fraudStaleEpochsHost{}
	_ fraudadmin.MLSnapshotHost            = fraudMLSnapshotHost{}
	_ fraudadmin.MLShadowDeltaSnapshotHost = fraudMLShadowDeltaHost{}
	_ fraudadmin.MLSyncHost                = (*Service)(nil)
)

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

var _ opsadmin.OpsMetricScraperHost = (*Service)(nil)

func (s *Service) startOpsMetricScraper(ctx context.Context, scrapeURL string) {
	opsadmin.StartMetricScraper(s, ctx, scrapeURL)
}

func NewManagementOpsReader(svc *Service) opsadmin.ManagementOpsReader {
	if svc == nil {
		return nil
	}
	var clickhouseQuery *database.ClickHouseQuery
	if svc.ClickHouseQuery() != nil {
		clickhouseQuery = svc.ClickHouseQuery()
	}
	return opsadmin.NewReader(opsadmin.ReaderDeps{
		Pool:        svc.GetPool(),
		RedisShards: svc.redisShards,
		Config:      svc.cfg,
		GetShardHealth: func(ctx context.Context) (opsadmin.ShardHealthReport, error) {
			report, err := svc.GetShardHealth(ctx)
			return report, err
		},
		ListReconRuns: svc.ListReconRuns,
		BuildStackHealthSnapshot: func(ctx context.Context) (opsadmin.StackHealthSnapshot, error) {
			return svc.BuildStackHealthSnapshot(ctx)
		},
		ClickHouseQuery: clickhouseQuery,
	})
}

func newOpsReader(svc *Service) opsadmin.ManagementOpsReader {
	return NewManagementOpsReader(svc)
}

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
	OpsAlerter               = opsadmin.OpsAlerter
	AlertmanagerAlert        = opsadmin.AlertmanagerAlert
	AlertmanagerWebhook      = opsadmin.AlertmanagerWebhook
)

type FanOutResult[T any] = opsadmin.FanOutResult[T]

type FanOutSource[T any] = opsadmin.FanOutSource[T]

var NewFanOutCollector = opsadmin.NewFanOutCollector

var (
	NewOpsAlerter               = opsadmin.NewOpsAlerter
	NewAlertmanagerWebhook      = opsadmin.NewAlertmanagerWebhook
	FormatAlertmanagerAlert     = opsadmin.FormatAlertmanagerAlert
	FormatAlertmanagerAlertText = opsadmin.FormatAlertmanagerAlertText
)

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

func (s *Service) StartFilterRejectRollupWorker(ctx context.Context, scrapeURL string) {
	if s == nil || s.GetPool() == nil || s.clickhouseQuery == nil {
		slog.Warn("filter reject rollup worker not started: postgres or clickhouse unavailable")
		return
	}
	w := opsadmin.NewFilterRejectRollupWorker(s.GetPool(), s.clickhouseQuery, scrapeURL)
	w.SetEdgeFetcher(func(ctx context.Context) (map[string]uint64, error) {
		panel, err := opsadmin.FetchEdgeMetrics(ctx)
		if err != nil {
			return nil, err
		}
		return panel.Blocked, nil
	})
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
}
