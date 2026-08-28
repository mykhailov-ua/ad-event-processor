package controlplane

import (
	"context"
	"fmt"

	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/fraud"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
