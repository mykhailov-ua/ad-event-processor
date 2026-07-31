package controlplane

import (
	"context"
)

func (worker *OutboxWorker) handleReloadRtbCatalog(ctx context.Context, payload []byte) error {
	_ = payload
	return worker.svc.PublishRtbCatalogReload(ctx)
}
