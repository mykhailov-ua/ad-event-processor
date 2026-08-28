package controlplane

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/fraud"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/shardadmin"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

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

func (s *Service) GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignFraudConfigDTO, error) {
	out, err := fraudadmin.GetCampaignFraudConfig(ctx, fraudCampaignConfigHost{svc: s}, campaignID)
	return out, mapFraudadminErr(err)
}

func (s *Service) PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req campaign.PreviewCampaignFraudRequest) (campaign.CampaignFraudPreviewDTO, error) {
	out, err := fraudadmin.PreviewCampaignFraudImpact(ctx, fraudCampaignConfigHost{svc: s}, campaignID, req)
	return out, mapFraudadminErr(err)
}

func (s *Service) UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, upd campaign.PatchCampaignFraudRequest) (campaign.CampaignFraudConfigDTO, error) {
	out, err := fraudadmin.UpdateCampaignFraudConfig(ctx, fraudCampaignConfigHost{svc: s}, campaignID, upd)
	return out, mapFraudadminErr(err)
}

func (s *Service) ApplyFraudScoringOverride(ctx context.Context, req fraudadmin.FraudScoringOverrideRequest) error {
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
