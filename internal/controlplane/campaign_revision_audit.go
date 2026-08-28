package controlplane

import (
	"context"
	"strings"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type auditRevisionConflictChange struct {
	ExpectedRevision string `json:"expected_revision"`
	ServerRevision   string `json:"server_revision"`
}

func (s *Service) AuditCampaignRevisionConflict(ctx context.Context, campaignID uuid.UUID, expectedRevision string) {
	if s == nil || s.GetPool() == nil {
		return
	}
	current, err := s.GetCampaign(ctx, campaignID)
	if err != nil {
		return
	}
	expectedRevision = strings.TrimSpace(expectedRevision)
	_ = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "CAMPAIGN_REVISION_CONFLICT", "campaign", &campaignID, auditRevisionConflictChange{
			ExpectedRevision: expectedRevision,
			ServerRevision:   campaignRevision(current.UpdatedAt),
		}, nil)
		return nil
	})
}
