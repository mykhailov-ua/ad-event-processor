package platformadmin

import (
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func optionalAuditText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func optionalAuditUUID(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{}
	}
	return domain.ToUUID(id)
}

func AuditFilterSQLParams(filter AuditLogFilter) db.CountAuditLogsFilteredParams {
	return auditFilterSQLParams(filter)
}

func AuditListSQLParams(filter AuditLogFilter, limit, offset int32) db.ListAuditPaginatedFilteredParams {
	return auditListSQLParams(filter, limit, offset)
}

func auditFilterSQLParams(filter AuditLogFilter) db.CountAuditLogsFilteredParams {
	return db.CountAuditLogsFilteredParams{
		AdminID:    optionalAuditUUID(filter.AdminID),
		TargetID:   optionalAuditUUID(filter.TargetID),
		Action:     optionalAuditText(filter.Action),
		AuthSource: optionalAuditText(filter.AuthSource),
		ApiKeyID:   optionalAuditText(filter.APIKeyID),
	}
}

func auditListSQLParams(filter AuditLogFilter, limit, offset int32) db.ListAuditPaginatedFilteredParams {
	base := auditFilterSQLParams(filter)
	return db.ListAuditPaginatedFilteredParams{
		Limit:      limit,
		Offset:     offset,
		AdminID:    base.AdminID,
		TargetID:   base.TargetID,
		Action:     base.Action,
		AuthSource: base.AuthSource,
		ApiKeyID:   base.ApiKeyID,
	}
}
