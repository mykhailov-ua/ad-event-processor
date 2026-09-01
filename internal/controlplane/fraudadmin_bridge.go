package controlplane

import (
	"context"
	"errors"
	"fmt"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/fraud"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

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

func actorUserID(ctx context.Context) uuid.UUID {
	if u, ok := GetUser(ctx); ok {
		return u.UserID
	}
	return uuid.Nil
}

var (
	_ fraudadmin.BlacklistJanitorHost      = (*Service)(nil)
	_ fraudadmin.LabelsHost                = (*Service)(nil)
	_ fraudadmin.DecisionsHost             = (*Service)(nil)
	_ fraudadmin.OverridesHost             = (*Service)(nil)
	_ fraudadmin.CampaignConfigHost        = (*Service)(nil)
	_ fraudadmin.PresetsHost               = (*Service)(nil)
	_ fraudadmin.StaleEpochsHost           = (*Service)(nil)
	_ fraudadmin.MLSnapshotHost            = (*Service)(nil)
	_ fraudadmin.MLShadowDeltaSnapshotHost = (*Service)(nil)
	_ fraudadmin.MLSyncHost                = (*Service)(nil)
)

func (s *Service) LabelsPool() *pgxpool.Pool                      { return s.GetPool() }
func (s *Service) DecisionsPool() *pgxpool.Pool                   { return s.GetPool() }
func (s *Service) OverridesPool() *pgxpool.Pool                   { return s.GetPool() }
func (s *Service) ConfigPool() *pgxpool.Pool                      { return s.GetPool() }
func (s *Service) PresetsPool() *pgxpool.Pool                     { return s.GetPool() }
func (s *Service) StaleEpochsPool() *pgxpool.Pool                 { return s.GetPool() }
func (s *Service) SnapshotPool() *pgxpool.Pool                    { return s.GetPool() }
func (s *Service) DecisionsClickHouse() *database.ClickHouseQuery { return s.clickhouseQuery }
func (s *Service) ConfigClickHouse() *database.ClickHouseQuery    { return s.clickhouseQuery }
func (s *Service) FraudExplainLiveScoreEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.FraudScoring.ExplainLiveScore
}

func (s *Service) FraudExplainScorer(ctx context.Context) (fraud.Scorer, error) {
	return s.fraudExplainScorer()
}
func (s *Service) OverrideActorID(ctx context.Context) uuid.UUID { return actorUserID(ctx) }
func (s *Service) ConfigActorID(ctx context.Context) uuid.UUID   { return actorUserID(ctx) }
func (s *Service) PresetActorID(ctx context.Context) uuid.UUID   { return actorUserID(ctx) }
func (s *Service) OverrideAuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any) {
	s.AuditLog(ctx, q, adminID, action, targetType, targetID, changes, metadata)
}

func (s *Service) OverrideHashIP(ip string) ([16]byte, error) {
	if s == nil || s.cfg == nil {
		return [16]byte{}, fmt.Errorf("piihash: service not configured")
	}
	return fraudadmin.HashIP(s.cfg.PIISaltVersion, string(s.cfg.PIISaltHex), string(s.cfg.TokenSymmetricKey), ip)
}

func (s *Service) OverrideEnqueueClearBoost(ctx context.Context, q db.Querier, campaignID string) error {
	payload, err := coldpath.MarshalOutbox(outbox.FraudThreatPayload{Action: "boost", CampaignID: campaignID})
	if err != nil {
		return err
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "ML_SCORE_BOOST", Payload: payload})
	return err
}

func (s *Service) OverrideEnqueueBlacklistRemove(ctx context.Context, q db.Querier, ip string) error {
	payload, err := coldpath.MarshalOutbox(outbox.BlacklistPayload{Action: "remove", IP: ip, Reason: "fraud"})
	if err != nil {
		return fmt.Errorf("marshal blacklist outbox payload: %w", err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "UPDATE_BLACKLIST", Payload: payload})
	return err
}

func (s *Service) ConfigAuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, campaignID uuid.UUID, changes fraudadmin.CampaignFraudAuditChange) {
	s.AuditLog(ctx, q, adminID, "UPDATE_CAMPAIGN_FRAUD", "campaign", &campaignID, platformadmin.AuditCampaignFraudChange{
		FraudThresholdPass: changes.FraudThresholdPass, FraudThresholdSuspect: changes.FraudThresholdSuspect,
		FraudThresholdIVT: changes.FraudThresholdIVT, FraudThresholdBlock: changes.FraudThresholdBlock,
		SilentRejectEnabled: changes.SilentRejectEnabled, BehaviorFlags: changes.BehaviorFlags,
		CanvasRetestEnabled: changes.CanvasRetestEnabled, CgnatIPPolicyEnabled: changes.CgnatIPPolicyEnabled,
		AcceptLangGeoEnabled: changes.AcceptLangGeoEnabled, JSONSerializationEnabled: changes.JSONSerializationEnabled,
	}, nil)
}

func (s *Service) ConfigResolvePresetThresholds(ctx context.Context, name string) (uint8, uint8, uint8, uint8, error) {
	pass, suspect, ivt, block, err := fraudadmin.ResolvePresetThresholds(ctx, s.GetPool(), name)
	return pass, suspect, ivt, block, mapFraudadminErr(err)
}

func (s *Service) ConfigEnqueueUpdateCampaignFraud(ctx context.Context, q db.Querier, campaignID uuid.UUID) error {
	payload, err := coldpath.MarshalOutbox(outbox.CampaignIDPayload{CampaignID: campaignID.String()})
	if err != nil {
		return fmt.Errorf("marshal update campaign fraud outbox payload: %w", err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "UPDATE_CAMPAIGN_FRAUD", Payload: payload})
	return err
}

func (s *Service) PresetAuditUpdate(ctx context.Context, q db.Querier, adminID uuid.UUID, name string, pass, suspect, ivt, block uint8) {
	s.AuditLog(ctx, q, adminID, "UPDATE_FRAUD_POLICY_PRESET", "system", nil, map[string]any{
		"name": name, "pass": pass, "suspect": suspect, "ivt": ivt, "block": block,
	}, nil)
}

func (s *Service) SnapshotRedisShards() []redis.UniversalClient {
	if s == nil {
		return nil
	}
	return s.redisShards
}
func (s *Service) StaleEpochsRedisShards() []redis.UniversalClient { return s.SnapshotRedisShards() }
func (s *Service) StaleEpochsUpdateSettings(ctx context.Context, settings map[string]string) error {
	return s.UpdateSettings(ctx, settings)
}

func (s *Service) BlacklistJanitorAlerter() fraudadmin.BlacklistJanitorAlerter {
	if s == nil {
		return nil
	}
	return s.alerter
}

func (s *Service) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]fraudadmin.MLManualLabelDTO, int64, error) {
	return fraudadmin.NewLabels(s).ListMLManualLabelsForCustomer(ctx, customerID, limit, offset)
}

func (s *Service) GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (campaign.CampaignFraudConfigDTO, error) {
	out, err := fraudadmin.GetCampaignFraudConfig(ctx, s, campaignID)
	return out, mapFraudadminErr(err)
}

func (s *Service) PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req campaign.PreviewCampaignFraudRequest) (campaign.CampaignFraudPreviewDTO, error) {
	out, err := fraudadmin.PreviewCampaignFraudImpact(ctx, s, campaignID, req)
	return out, mapFraudadminErr(err)
}

func (s *Service) UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, upd campaign.PatchCampaignFraudRequest) (campaign.CampaignFraudConfigDTO, error) {
	out, err := fraudadmin.UpdateCampaignFraudConfig(ctx, s, campaignID, upd)
	return out, mapFraudadminErr(err)
}

func (s *Service) ApplyFraudScoringOverride(ctx context.Context, req fraudadmin.FraudScoringOverrideRequest) error {
	return mapFraudadminErr(fraudadmin.ApplyFraudScoringOverride(ctx, s, req))
}

func (s *Service) CheckAndHandleStaleEpochs(ctx context.Context) error {
	return fraudadmin.CheckAndHandleStaleEpochs(ctx, s)
}
