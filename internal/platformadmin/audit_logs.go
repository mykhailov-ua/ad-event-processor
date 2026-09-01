package platformadmin

import (
	"context"
	"encoding/json"
	"net"
	"regexp"
	"strings"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
)

type AuditLogsHost interface {
	ListAuditLogRows(ctx context.Context, limit, offset int32) ([]db.AdminAuditLog, int64, error)
}

var emailPIIPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

func ListAuditLogs(ctx context.Context, host AuditLogsHost, limit, offset int32, redactPII bool) ([]AuditLogDTO, int64, error) {
	rows, total, err := host.ListAuditLogRows(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return mapAuditRows(rows, redactPII), total, nil
}

func mapAuditRows(rows []db.AdminAuditLog, redactPII bool) []AuditLogDTO {
	out := make([]AuditLogDTO, len(rows))
	for i, row := range rows {
		changes := row.Changes
		metadata := row.Metadata
		if redactPII {
			changes = redactJSONPII(changes)
			metadata = redactJSONPII(metadata)
		}
		dto := AuditLogDTO{
			ID:         row.ID,
			Action:     row.Action,
			TargetType: row.TargetType,
			Changes:    changes,
			Metadata:   metadata,
			IsMasked:   row.IsMasked,
		}
		if row.AdminID.Valid {
			dto.AdminID = uuid.UUID(row.AdminID.Bytes).String()
		}
		if row.TargetID.Valid {
			dto.TargetID = uuid.UUID(row.TargetID.Bytes).String()
		}
		if row.CreatedAt.Valid {
			dto.CreatedAt = row.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z")
			dto.CreatedAtDisplay = coldpath.RFC3339Display(dto.CreatedAt)
		}
		out[i] = dto
	}
	return out
}

func redactJSONPII(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return []byte(redactStringPII(string(raw)))
	}
	redactValuePII(&v)
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

func redactValuePII(v *any) {
	switch val := (*v).(type) {
	case map[string]any:
		for k, child := range val {
			lk := strings.ToLower(k)
			switch {
			case lk == "email" || strings.Contains(lk, "email"):
				val[k] = "[REDACTED_EMAIL]"
			case lk == "ip" || lk == "ip_address" || strings.HasSuffix(lk, "_ip"):
				val[k] = "[REDACTED_IP]"
			default:
				childCopy := child
				redactValuePII(&childCopy)
				val[k] = childCopy
			}
		}
	case []any:
		for i := range val {
			redactValuePII(&val[i])
		}
	case string:
		*v = redactStringPII(val)
	}
}

func redactStringPII(s string) string {
	if net.ParseIP(s) != nil {
		return "[REDACTED_IP]"
	}
	if emailPIIPattern.MatchString(s) {
		return emailPIIPattern.ReplaceAllString(s, "[REDACTED_EMAIL]")
	}
	return s
}
