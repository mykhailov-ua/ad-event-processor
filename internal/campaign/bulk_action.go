package campaign

import (
	"context"

	"github.com/google/uuid"
)

func RunBulkCampaignAction(
	ctx context.Context,
	action string,
	ids []uuid.UUID,
	reason string,
	pause func(context.Context, uuid.UUID, string) error,
	resume func(context.Context, uuid.UUID, string) error,
	archive func(context.Context, uuid.UUID, string) error,
) map[uuid.UUID]error {
	out := make(map[uuid.UUID]error, len(ids))
	for _, id := range ids {
		var err error
		switch action {
		case "pause":
			err = pause(ctx, id, reason)
		case "resume":
			err = resume(ctx, id, reason)
		case "archive":
			err = archive(ctx, id, reason)
		default:
			err = ErrValidationf("unsupported bulk action")
		}
		if err != nil {
			out[id] = err
		}
	}
	return out
}
