package campaign

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
)

func maskLevelFromContext(ctx context.Context) authz.MaskLevel {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return authz.MaskMasked
	}
	return snap.Mask
}
