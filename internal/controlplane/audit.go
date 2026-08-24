package controlplane

import (
	"context"
	"log/slog"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"

	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any) {
	s.auditLogMasked(ctx, q, adminID, action, targetType, targetID, changes, metadata, authz.IsMaskedMutation(ctx))
}

func (s *Service) auditLogMasked(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any, isMasked bool) {
	changesJSON, err := coldpath.MarshalJSON(changes)
	if err != nil {
		slog.Error("audit marshal changes failed", "error", err, "admin_id", adminID, "action", action)
		changesJSON = []byte("{}")
	}
	metadataJSON, err := coldpath.MarshalJSON(metadata)
	if err != nil {
		slog.Error("audit marshal metadata failed", "error", err, "admin_id", adminID, "action", action)
		metadataJSON = []byte("{}")
	}

	var tid pgtype.UUID
	if targetID != nil {
		tid = domain.ToUUID(*targetID)
	}

	if q == nil {
		q = db.New(s.GetPool())
	}

	_, err = q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		AdminID:    domain.ToUUID(adminID),
		Action:     action,
		TargetType: targetType,
		TargetID:   tid,
		Changes:    changesJSON,
		Metadata:   metadataJSON,
		IsMasked:   isMasked,
	})
	if err != nil {
		slog.Error("failed to write audit log", "error", err, "admin_id", adminID, "action", action)
	}
}

func (s *Service) RunAuditCleaner(ctx context.Context, retention Days) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanOldLogs(ctx, retention)
		}
	}
}

type Days int

func (s *Service) cleanOldLogs(ctx context.Context, retention Days) {
	threshold := time.Now().AddDate(0, 0, -int(retention))
	err := db.New(s.GetPool()).CleanupAuditLogs(ctx, pgtype.Timestamptz{Time: threshold, Valid: true})
	if err != nil {
		slog.Error("failed to cleanup audit logs", "error", err)
	} else {
		slog.Info("audit logs cleaned up", "older_than", threshold.Format(time.RFC3339))
	}
}
