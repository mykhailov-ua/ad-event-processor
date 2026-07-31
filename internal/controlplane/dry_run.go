package controlplane

import (
	"net/http"
)

type MutationPreview struct {
	DryRun      bool           `json:"dry_run"`
	Action      string         `json:"action"`
	WouldChange map[string]any `json:"would_change"`
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
