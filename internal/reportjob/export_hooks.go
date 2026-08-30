package reportjob

import (
	"context"
	"strings"
)

var (
	// Wired by controlplane bridge; labels audit columns on CH export when deps set.
	ExportActorLabel   func(context.Context) string
	ExportDeploymentID func() string
)

func exportActorLabel(ctx context.Context) string {
	if ExportActorLabel != nil {
		return strings.TrimSpace(ExportActorLabel(ctx))
	}
	return ""
}
