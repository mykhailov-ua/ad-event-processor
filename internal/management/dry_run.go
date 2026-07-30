package management

import (
	"net/http"

	"github.com/google/uuid"
)

type MutationPreview struct {
	DryRun      bool           `json:"dry_run"`
	Action      string         `json:"action"`
	WouldChange map[string]any `json:"would_change"`
}

func (h *Handler) logDryRunAudit(r *http.Request, action, targetType string, targetID *uuid.UUID, wouldChange map[string]any) {
	if h == nil || h.svc == nil {
		return
	}
	meta := map[string]any{"dry_run": true}
	for k, v := range wouldChange {
		meta[k] = v
	}
	var uid uuid.UUID
	if u, ok := GetUser(r.Context()); ok {
		uid = u.UserID
	}
	h.svc.AuditLog(r.Context(), nil, uid, action, targetType, targetID, meta, nil)
}

func ParseDryRun(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.Header.Get("X-Dry-Run") == "1" {
		return true
	}
	return r.URL.Query().Get("dry_run") == "1"
}
