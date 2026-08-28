package reportjob

import (
	"context"
	"strings"
)

var (
	ExportActorLabel   func(context.Context) string
	ExportDeploymentID func() string
)

func exportActorLabel(ctx context.Context) string {
	if ExportActorLabel != nil {
		return strings.TrimSpace(ExportActorLabel(ctx))
	}
	return ""
}
