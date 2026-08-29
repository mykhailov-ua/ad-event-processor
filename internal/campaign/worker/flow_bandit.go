package worker

import (
	"context"
	"time"

	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
)

type flowBanditAdapter struct {
	host BanditDeliveryHost
}

func (a flowBanditAdapter) MABLookbackDays() int {
	return a.host.MABLookbackDays()
}

func (a flowBanditAdapter) QueryFlowBanditStats(ctx context.Context, from, to time.Time) (
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	error,
) {
	return a.host.QueryFlowBanditStats(ctx, from, to)
}

var _ flow.BanditHost = flowBanditAdapter{}
