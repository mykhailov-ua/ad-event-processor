package filter

import (
	"context"

	"ad-event-processor/internal/domain"
)

type DBHealthChecker interface {
	Ping(ctx context.Context) error
}

type DebitFilter interface {
	EventFilter
	StreamDeferredToProducer() bool
	SetDeferStreamToProducer(deferWrite bool)
	ClickAmountMicro() int64
	ImpressionAmountMicro() int64
	LocalQuantaFullSkipEligible(evt *domain.Event, camp *domain.Campaign) bool
	RollbackDebit(ctx context.Context, evt *domain.Event, camp *domain.Campaign, debitAmount int64, isLocalQuanta bool)
	SetSkipBudgetDebit(skip bool)
}
