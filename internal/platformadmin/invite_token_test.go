package platformadmin_test

import (
	"context"
	"testing"

	"ad-event-processor/internal/platformadmin"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestInviteToken_roundTrip(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	token := "invite-token-abc"
	payload := platformadmin.TeamInvitePayload{
		UserID:     uuid.New(),
		CustomerID: uuid.New(),
		Email:      "member@example.com",
	}
	require.NoError(t, platformadmin.StoreTeamInvite(ctx, rdb, token, payload))

	got, err := platformadmin.LoadTeamInvite(ctx, rdb, token)
	require.NoError(t, err)
	require.Equal(t, payload.UserID, got.UserID)
	require.Equal(t, payload.CustomerID, got.CustomerID)
	require.Equal(t, payload.Email, got.Email)

	require.NoError(t, platformadmin.DeleteTeamInvite(ctx, rdb, token))
	_, err = platformadmin.LoadTeamInvite(ctx, rdb, token)
	require.ErrorIs(t, err, platformadmin.ErrInviteInvalid)
}
