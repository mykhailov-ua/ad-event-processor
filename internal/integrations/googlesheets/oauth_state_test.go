package googlesheets

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOAuthState_verifyReturnsUser_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("01234567890123456789012345678901")
	userID := uuid.New()
	state, err := issueOAuthState(secret, userID)
	require.NoError(t, err)
	got, err := verifyOAuthStateReturnUser(secret, state)
	require.NoError(t, err)
	require.Equal(t, userID, got)
}

func TestOAuthState_issueAndVerify_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("01234567890123456789012345678901")
	userID := uuid.New()
	state, err := issueOAuthState(secret, userID)
	require.NoError(t, err)
	require.NotEmpty(t, state)
	require.NoError(t, verifyOAuthState(secret, state, userID))
}

func TestOAuthState_rejectsWrongUser_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("01234567890123456789012345678901")
	userID := uuid.New()
	state, err := issueOAuthState(secret, userID)
	require.NoError(t, err)
	require.Error(t, verifyOAuthState(secret, state, uuid.New()))
}

func TestOAuthState_rejectsTamperedSignature_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("01234567890123456789012345678901")
	userID := uuid.New()
	state, err := issueOAuthState(secret, userID)
	require.NoError(t, err)
	require.Error(t, verifyOAuthState([]byte("wrong-secret-key-012345678901"), state, userID))
}

func TestOAuthState_rejectsExpiredState_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("01234567890123456789012345678901")
	userID := uuid.New()
	nonce, err := uuid.NewV7()
	require.NoError(t, err)
	exp := time.Now().UTC().Add(-time.Minute).Unix()
	payload := fmt.Sprintf("%s|%s|%d", userID.String(), nonce.String(), exp)
	sig := signState(secret, payload)
	state := base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig))
	require.ErrorIs(t, verifyOAuthState(secret, state, userID), ErrStateExpired)
}

func TestConnectURL_includesScopeAndState(t *testing.T) {
	t.Parallel()
	got, err := ConnectURL(Config{
		ClientID:  "client-id",
		PublicURL: "https://admin.example",
	}, "state-token")
	require.NoError(t, err)
	require.Contains(t, got, "client_id=client-id")
	require.Contains(t, got, "state=state-token")
	require.Contains(t, got, url.QueryEscape(OAuthScope))
	require.Contains(t, got, "access_type=offline")
}

func TestRedirectURI_usesPublicURL(t *testing.T) {
	t.Parallel()
	require.Equal(
		t,
		"https://admin.example/api/v1/integrations/google-sheets/callback",
		RedirectURI("https://admin.example"),
	)
}
