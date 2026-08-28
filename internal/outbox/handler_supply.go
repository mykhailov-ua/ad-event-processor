package outbox

import "context"

func (w *Worker) handleUpdateSupplyFiles(ctx context.Context, payload []byte) error {
	_ = payload
	return w.host.ExportSupplyFiles(ctx)
}