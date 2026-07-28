package management

import (
	"net/http"

	"github.com/google/uuid"
)

// MutationPreview describes side effects that would occur on a live admin mutation (GAP-RTB-12b).
type MutationPreview struct {
	DryRun      bool           `json:"dry_run"`
	Action      string         `json:"action"`
	WouldChange map[string]any `json:"would_change"`
}

// logDryRunAudit records a dry-run mutation in audit_logs with dry_run=true metadata (GAP-RTB-12b).
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

// ParseDryRun returns true when the client requests a simulation-only mutation.
// Supported: header X-Dry-Run: 1 or query ?dry_run=1.
func ParseDryRun(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.Header.Get("X-Dry-Run") == "1" {
		return true
	}
	return r.URL.Query().Get("dry_run") == "1"
}
