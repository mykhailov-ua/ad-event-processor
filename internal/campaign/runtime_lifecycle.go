package campaign

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listCampaignEvents(
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignID uuid.UUID,
	limit, offset int32,
) ([]CampaignEventDTO, int64, error) {
	if pool == nil {
		return nil, 0, fmt.Errorf("service unavailable")
	}
	q := db.New(pool)
	cid := domain.ToUUID(campaignID)
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountCampaignEvents(ctx, cid) },
		func() ([]db.ListCampaignEventsRow, error) {
			return q.ListCampaignEvents(ctx, db.ListCampaignEventsParams{
				CampaignID: cid,
				Limit:      limit,
				Offset:     offset,
			})
		},
		campaignEventToDTO,
	)
}

func campaignEventToDTO(row db.ListCampaignEventsRow) CampaignEventDTO {
	var ip, ua, userID string
	if row.IpAddress.Valid {
		ip = row.IpAddress.String
	}
	if row.UserAgent.Valid {
		ua = row.UserAgent.String
	}
	if row.UserID.Valid {
		userID = row.UserID.String
	}
	createdAt := ""
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return CampaignEventDTO{
		ClickID:   row.ClickID,
		EventType: row.EventType,
		UserID:    userID,
		IP:        ip,
		UserAgent: ua,
		Payload:   json.RawMessage(row.Payload),
		CreatedAt: createdAt,
	}
}

func pauseCampaign(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID, reason string) error {
	if pool == nil || fx == nil {
		return errServiceUnavailable()
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}
		if err := assertMediaBuyerCampaignAccess(ctx, camp); err != nil {
			return err
		}
		if camp.Status == db.CampaignStatusTypePAUSED {
			return nil
		}
		if camp.Status != db.CampaignStatusTypeACTIVE {
			return fmt.Errorf("%w in status %s", ErrCampaignCannotBePaused, camp.Status)
		}
		if _, err := q.PauseCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
			return err
		}
		if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: camp.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypePAUSED,
			Reason:     pgtype.Text{String: reason, Valid: reason != ""},
		}); err != nil {
			return err
		}
		adminID := uuid.Nil
		if user, ok := authz.GetUser(ctx); ok {
			adminID = user.UserID
		}
		fx.AuditLog(ctx, q, adminID, "PAUSE_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
		return fx.EnqueueCampaignOutbox(ctx, q, "PAUSE_CAMPAIGN", campaignID, camp.BudgetLimit)
	})
}

func resumeCampaign(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID, reason string, publishForce bool) error {
	if pool == nil || fx == nil {
		return errServiceUnavailable()
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}
		if err := assertMediaBuyerCampaignAccess(ctx, camp); err != nil {
			return err
		}
		if camp.Status != db.CampaignStatusTypePAUSED {
			return ErrCampaignNotPaused
		}
		now := time.Now()
		var startAt, endAt *time.Time
		if camp.StartAt.Valid {
			startAt = &camp.StartAt.Time
		}
		if camp.EndAt.Valid {
			endAt = &camp.EndAt.Time
		}
		if ResolveScheduleStatus(now, startAt, endAt) != db.CampaignStatusTypeACTIVE {
			return ErrCampaignOutsideSchedule
		}
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, camp, publishForce); err != nil {
			return err
		}
		if _, err := q.ResumeCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
			return err
		}
		if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: camp.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypeACTIVE,
			Reason:     pgtype.Text{String: reason, Valid: reason != ""},
		}); err != nil {
			return err
		}
		adminID := uuid.Nil
		if user, ok := authz.GetUser(ctx); ok {
			adminID = user.UserID
		}
		fx.AuditLog(ctx, q, adminID, "RESUME_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
		return fx.EnqueueCampaignOutbox(ctx, q, "RESUME_CAMPAIGN", campaignID, camp.BudgetLimit)
	})
}

type auditReasonChange struct {
	Reason string `json:"reason"`
}
