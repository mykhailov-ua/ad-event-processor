package controlplane

import (
	"context"
	"net/http"

	"ad-event-processor/internal/campaign"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type campaignStatusPatchOpts struct {
	publishForce bool
}

type CampaignPublishBlockedError = campaign.CampaignPublishBlockedError

var ErrCampaignPublishBlocked = campaign.ErrCampaignPublishBlocked

func parsePublishForceQuery(raw string) bool {
	return campaign.ParsePublishForceQuery(raw)
}

func (s *Service) EvaluateCampaignPublish(ctx context.Context, campaignID uuid.UUID) (CampaignPublishCheckDTO, error) {
	return s.CampaignRuntime().EvaluateCampaignPublish(ctx, campaignID)
}

func (s *Service) PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (CampaignDTO, error) {
	return s.CampaignRuntime().PublishCampaign(ctx, campaignID, force)
}

func (s *Service) enforceCampaignPublishGate(ctx context.Context, campaignID uuid.UUID, row db.Campaign, force bool) error {
	if force {
		return s.auditCampaignPublishForce(ctx, campaignID)
	}
	blocked, err := campaign.CollectPublishBlocked(ctx, s, campaignID, row)
	if err != nil {
		return err
	}
	if blocked != nil {
		return blocked
	}
	return nil
}

func (s *Service) auditCampaignPublishForce(ctx context.Context, campaignID uuid.UUID) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		var uid uuid.UUID
		if user, ok := GetUser(ctx); ok {
			uid = user.UserID
		}
		s.AuditLog(ctx, q, uid, "PUBLISH_FORCE", "campaign", &campaignID, auditReasonChange{Reason: "publish_gate_bypass"}, nil)
		return nil
	})
}

func writeCampaignPublishError(w http.ResponseWriter, err error) {
	campaign.WriteCampaignPublishError(w, err, func(w http.ResponseWriter, err error) {
		writeServiceError(w, err)
	})
}

var _ campaign.CampaignReader = (*Service)(nil)
