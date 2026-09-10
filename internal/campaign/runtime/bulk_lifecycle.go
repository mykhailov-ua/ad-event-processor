package runtime

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (r *Runtime) BulkCampaignAction(ctx context.Context, action string, ids []uuid.UUID, reason string) map[uuid.UUID]error {
	if r == nil || r.effects == nil {
		out := make(map[uuid.UUID]error, len(ids))
		for _, id := range ids {
			out[id] = campaign.ErrServiceUnavailable()
		}
		return out
	}
	return bulkCampaignLifecycle(ctx, r.PoolOrNil(), r.effects, action, ids, reason)
}

func bulkCampaignLifecycle(
	ctx context.Context,
	pool *pgxpool.Pool,
	fx campaign.Effects,
	action string,
	ids []uuid.UUID,
	reason string,
) map[uuid.UUID]error {
	errByID := make(map[uuid.UUID]error)
	if pool == nil || fx == nil {
		for _, id := range ids {
			errByID[id] = campaign.ErrServiceUnavailable()
		}
		return errByID
	}
	unique := sortCampaignIDs(ids)
	if len(unique) == 0 {
		return errByID
	}

	txErr := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		pgIDs := make([]pgtype.UUID, len(unique))
		for i, id := range unique {
			pgIDs[i] = domain.ToUUID(id)
		}
		rows, err := q.ListCampaignsForUpdate(ctx, pgIDs)
		if err != nil {
			return err
		}
		byID := make(map[uuid.UUID]db.Campaign, len(rows))
		for i := range rows {
			byID[uuid.UUID(rows[i].ID.Bytes)] = rows[i]
		}
		adminID := adminIDFromCtx(ctx)
		for _, id := range unique {
			camp, ok := byID[id]
			if !ok {
				errByID[id] = campaign.ErrCampaignNotFound
				continue
			}
			var opErr error
			switch action {
			case "pause":
				opErr = pauseCampaignLocked(ctx, pool, q, fx, camp, id, reason, adminID)
			case "resume":
				opErr = resumeCampaignLocked(ctx, pool, q, fx, camp, id, reason, false, adminID)
			case "archive":
				opErr = archiveCampaignLocked(ctx, pool, q, fx, camp, reason)
			default:
				return fmt.Errorf("unsupported bulk action %q", action)
			}
			if opErr != nil {
				errByID[id] = opErr
			}
		}
		return nil
	})
	if txErr != nil {
		for _, id := range unique {
			if _, recorded := errByID[id]; !recorded {
				errByID[id] = mapCampaignStoreError(txErr)
			}
		}
	}
	return errByID
}

func sortCampaignIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		return bytes.Compare(out[i][:], out[j][:]) < 0
	})
	return out
}

func adminIDFromCtx(ctx context.Context) uuid.UUID {
	if user, ok := authz.GetUser(ctx); ok {
		return user.UserID
	}
	return uuid.Nil
}

func pauseCampaignLocked(
	ctx context.Context,
	pool *pgxpool.Pool,
	q *db.Queries,
	fx campaign.Effects,
	camp db.Campaign,
	campaignID uuid.UUID,
	reason string,
	adminID uuid.UUID,
) error {
	if err := campaign.AssertCampaignAccess(ctx, pool, camp); err != nil {
		return err
	}
	if camp.Status == db.CampaignStatusTypePAUSED {
		return nil
	}
	if camp.Status != db.CampaignStatusTypeACTIVE {
		return fmt.Errorf("%w in status %s", campaign.ErrCampaignCannotBePaused, camp.Status)
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
	fx.AuditLog(ctx, q, adminID, "PAUSE_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
	return fx.EnqueueCampaignOutbox(ctx, q, "PAUSE_CAMPAIGN", campaignID, camp.BudgetLimit)
}

func resumeCampaignLocked(
	ctx context.Context,
	pool *pgxpool.Pool,
	q *db.Queries,
	fx campaign.Effects,
	camp db.Campaign,
	campaignID uuid.UUID,
	reason string,
	publishForce bool,
	adminID uuid.UUID,
) error {
	if err := campaign.AssertCampaignAccess(ctx, pool, camp); err != nil {
		return err
	}
	if camp.Status != db.CampaignStatusTypePAUSED {
		return campaign.ErrCampaignNotPaused
	}
	now := time.Now()
	var startAt, endAt *time.Time
	if camp.StartAt.Valid {
		startAt = &camp.StartAt.Time
	}
	if camp.EndAt.Valid {
		endAt = &camp.EndAt.Time
	}
	if campaign.ResolveScheduleStatus(now, startAt, endAt) != db.CampaignStatusTypeACTIVE {
		return campaign.ErrCampaignOutsideSchedule
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
	fx.AuditLog(ctx, q, adminID, "RESUME_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
	return fx.EnqueueCampaignOutbox(ctx, q, "RESUME_CAMPAIGN", campaignID, camp.BudgetLimit)
}

func archiveCampaignLocked(
	ctx context.Context,
	pool *pgxpool.Pool,
	q *db.Queries,
	fx campaign.Effects,
	camp db.Campaign,
	reason string,
) error {
	if err := campaign.AssertCampaignAccess(ctx, pool, camp); err != nil {
		return err
	}
	return campaign.ArchiveCampaignStatus(ctx, fx, q, camp, reason)
}
