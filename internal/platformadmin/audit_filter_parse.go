package platformadmin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type AuditLogFilter struct {
	AdminID    uuid.UUID
	TargetID   uuid.UUID
	Action     string
	AuthSource string
	APIKeyID   string
}

func ParseAuditListFilter(r *http.Request) (AuditLogFilter, error) {
	q := r.URL.Query()
	filter := AuditLogFilter{}

	if raw := strings.TrimSpace(q.Get("admin_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return AuditLogFilter{}, fmt.Errorf("admin_id must be a UUID")
		}
		filter.AdminID = id
	}

	targetRaw := strings.TrimSpace(q.Get("target_id"))
	if targetRaw == "" {
		targetRaw = strings.TrimSpace(q.Get("campaign_id"))
	}
	if targetRaw != "" {
		id, err := uuid.Parse(targetRaw)
		if err != nil {
			return AuditLogFilter{}, fmt.Errorf("target_id must be a UUID")
		}
		filter.TargetID = id
	}

	filter.Action = strings.TrimSpace(q.Get("action"))

	authSource := strings.TrimSpace(q.Get("auth_source"))
	switch authSource {
	case "":
	case "session", "api_key":
		filter.AuthSource = authSource
	default:
		return AuditLogFilter{}, fmt.Errorf("auth_source must be session or api_key")
	}

	if raw := strings.TrimSpace(q.Get("api_key_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return AuditLogFilter{}, fmt.Errorf("api_key_id must be a UUID")
		}
		filter.APIKeyID = id.String()
	}

	return filter, nil
}
