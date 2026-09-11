package filter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestReserveOfferClick_enforcesDailyCap_holdout(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	offerID := uuid.New()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		ok, err := ReserveOfferClick(ctx, rdb, offerID, 3, 0, time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		require.True(t, ok)
	}
	ok, err := ReserveOfferClick(ctx, rdb, offerID, 3, 0, time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.False(t, ok)
}
