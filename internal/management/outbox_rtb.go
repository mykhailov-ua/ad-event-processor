package management

import (
	"context"

	"espx/pkg/coldpath"
)

func (w *OutboxWorker) handleReloadRtbCatalog(ctx context.Context, payload []byte) error {
	_ = coldpath.UnmarshalLenient[RtbCatalogReloadPayload](payload)
	return w.svc.PublishRtbCatalogReload(ctx)
}
