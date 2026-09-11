package googlesheets

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStore_tokenRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	encKey := []byte("01234567890123456789012345678901")
	st := NewStore(pool, encKey)
	ctx := context.Background()

	userID := uuid.New()

	expires := time.Now().UTC().Add(time.Hour)
	require.NoError(t, st.UpsertTokens(ctx, userID, "refresh-token", "access-token", expires, OAuthScope))

	connected, err := st.HasConnection(ctx, userID)
	require.NoError(t, err)
	require.True(t, connected)

	row, err := st.LoadTokenRow(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, "refresh-token", row.RefreshToken)
	require.Equal(t, "access-token", row.AccessToken)
	require.Equal(t, OAuthScope, row.Scopes)
	require.False(t, row.ExpiresAt.IsZero())

	require.NoError(t, st.DeleteConnection(ctx, userID))
	connected, err = st.HasConnection(ctx, userID)
	require.NoError(t, err)
	require.False(t, connected)
}
