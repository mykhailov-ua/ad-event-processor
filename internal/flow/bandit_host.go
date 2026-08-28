package flow

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EntityBanditStat struct {
	Clicks      int64
	Conversions int64
	Payout      float64
}

type BanditHost interface {
	MABLookbackDays() int
	QueryFlowBanditStats(ctx context.Context, from, to time.Time) (
		map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
		map[uuid.UUID]map[uuid.UUID]EntityBanditStat,
		error,
	)
}
