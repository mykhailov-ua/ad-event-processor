package authz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMergeAuditMetadata_session(t *testing.T) {
	userID := uuid.New()
	ctx := WithAuthenticatedUser(context.Background(), AuthenticatedUser{
		UserID:     userID,
		AuthSource: "session",
	})
	got := MergeAuditMetadata(ctx, map[string]any{"action": "patch"})
	meta, ok := got.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "session", meta["auth_source"])
	require.Equal(t, "patch", meta["action"])
	_, hasKey := meta["api_key_id"]
	require.False(t, hasKey)
}

func TestMergeAuditMetadata_apiKey(t *testing.T) {
	keyID := uuid.New()
	ctx := WithAuthenticatedUser(context.Background(), AuthenticatedUser{
		UserID:     uuid.New(),
		AuthSource: "api_key",
		APIKeyID:   keyID,
	})
	got := MergeAuditMetadata(ctx, nil)
	meta, ok := got.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "api_key", meta["auth_source"])
	require.Equal(t, keyID.String(), meta["api_key_id"])
}
