package reports

import "context"

var (
	ExportActorLabel   func(context.Context) string
	ExportDeploymentID func() string
)
