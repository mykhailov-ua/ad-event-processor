package outbox

import (
	"context"
	"fmt"

	"ad-event-processor/pkg/coldpath"
)

func (w *Worker) handleLanderPublished(ctx context.Context, payload []byte) error {
	if w == nil || w.host == nil {
		return fmt.Errorf("outbox worker unavailable")
	}
	if _, err := coldpath.UnmarshalStrict[LanderPublishedPayload](payload); err != nil {
		return err
	}
	return PublishFlowReload(ctx, w.host.RedisShards(), w.host.FlowReloadChannel())
}