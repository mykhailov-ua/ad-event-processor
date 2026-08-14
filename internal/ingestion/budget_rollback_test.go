package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix(t *testing.T) {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{"localhost:6379"},
	})
	defer func() { _ = rdb.Close() }()

	f := NewUnifiedFilter([]redis.UniversalClient{rdb}, nil, nil, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	stream := NewLocalQuantaStreamPublisher(LocalQuantaStreamPublisherConfig{
		Rdbs:           []redis.UniversalClient{rdb},
		StreamName:     "events",
		MaxLen:         1000,
		IdempotencyTTL: time.Hour,
	})
	defer stream.Close()

	f.SetLocalQuantaDeps(LocalQuantaDeps{Stream: stream})

	// Initially, the stream name should be "events"
	require.Equal(t, "events", stream.stream)

	// After setting defer stream to producer, the stream name must be "fcap:ignored" to prevent double writing
	f.SetDeferStreamToProducer(true)
	require.Equal(t, "fcap:ignored", stream.stream)

	// Setting it back should restore the stream name
	f.SetDeferStreamToProducer(false)
	require.Equal(t, "events", stream.stream)
}

func TestUnifiedFilter_RollbackDebit_LocalQuanta(t *testing.T) {
	ledger := NewLocalQuantaLedger()
	f := NewUnifiedFilter(nil, nil, nil, nil, 0, time.Minute, time.Hour, time.Hour, 100, 10, "events", 1000)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger})

	campID := uuid.New()
	campInfo := &domain.Campaign{
		ID:                campID,
		BudgetCampaignKey: "budget:" + campID.String(),
	}
	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "user-1",
		ClickID:    "click-1",
	}

	subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)

	// Credit some budget
	ledger.Credit(campID, 1000, 1000)
	require.Equal(t, int64(1000), ledger.Remaining(campID))

	// Spend some budget
	require.True(t, ledger.TrySpendDebit(campID, subSlot, 100))
	require.Equal(t, int64(900), ledger.Remaining(campID))

	// Rollback the spend
	f.RollbackDebit(context.Background(), evt, campInfo, 100, true)
	require.Equal(t, int64(1000), ledger.Remaining(campID))
}
