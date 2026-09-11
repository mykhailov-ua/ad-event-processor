package googlesheets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type callbackTestStore struct {
	upsertUserID uuid.UUID
	upsertCalled bool
}

func (m *callbackTestStore) HasConnection(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

func (m *callbackTestStore) OperatorAccountEmail(context.Context, uuid.UUID) (string, error) {
	return "", nil
}

func (m *callbackTestStore) UpsertTokens(_ context.Context, userID uuid.UUID, _, _ string, _ time.Time, _ string) error {
	m.upsertUserID = userID
	m.upsertCalled = true
	return nil
}

func (m *callbackTestStore) DeleteConnection(context.Context, uuid.UUID) error {
	return nil
}

func TestHTTPHandlers_getStatus_storeNil_notConfigured(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	h := &HTTPHandlers{
		Config: Config{ClientID: "client", ClientSecret: "secret"},
		ActorUserID: func(*http.Request) (uuid.UUID, bool) {
			return userID, true
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/google-sheets/status", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"connected":false`)
	require.Contains(t, w.Body.String(), `"not configured"`)
}

func TestHTTPHandlers_getStatus_notConfigured(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	h := &HTTPHandlers{
		Config: Config{},
		Store:  &Store{},
		ActorUserID: func(*http.Request) (uuid.UUID, bool) {
			return userID, true
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/google-sheets/status", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"connected":false`)
	require.Contains(t, w.Body.String(), `"not configured"`)
}

func TestHTTPHandlers_getConnect_notConfigured503(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	h := &HTTPHandlers{
		Config: Config{},
		Store:  &Store{},
		ActorUserID: func(*http.Request) (uuid.UUID, bool) {
			return userID, true
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/google-sheets/connect", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Contains(t, w.Body.String(), "NOT_CONFIGURED")
}

func TestHTTPHandlers_getCallback_rejectsInvalidState_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("01234567890123456789012345678901")
	userID := uuid.New()
	h := &HTTPHandlers{
		Config: Config{
			ClientID:     "client-id",
			ClientSecret: "secret",
			PublicURL:    "https://admin.example",
			StateSecret:  secret,
		},
		Store: &Store{},
		ActorUserID: func(*http.Request) (uuid.UUID, bool) {
			return userID, true
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/integrations/google-sheets/callback?code=abc&state=bad-state",
		http.NoBody,
	)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid oauth state")
}

func TestHTTPHandlers_getCallback_usesStateUser_notSession_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("01234567890123456789012345678901")
	stateUserID := uuid.New()
	state, err := issueOAuthState(secret, stateUserID)
	require.NoError(t, err)

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	mockStore := &callbackTestStore{}
	h := &HTTPHandlers{
		Config: Config{
			ClientID:     "client-id",
			ClientSecret: "secret",
			PublicURL:    "https://admin.example",
			StateSecret:  secret,
			TokenURL:     tokenSrv.URL,
		},
		Store: mockStore,
		ActorUserID: func(*http.Request) (uuid.UUID, bool) {
			return uuid.Nil, false
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/integrations/google-sheets/callback?code=auth-code&state="+state,
		http.NoBody,
	)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	require.Equal(t, "/integrations/google-sheets?google_sheets=connected", w.Header().Get("Location"))
	require.True(t, mockStore.upsertCalled)
	require.Equal(t, stateUserID, mockStore.upsertUserID)
}

func TestHTTPHandlers_getCallback_rejectsSessionUserMismatch_holdout(t *testing.T) {
	t.Parallel()
	secret := []byte("01234567890123456789012345678901")
	stateUserID := uuid.New()
	state, err := issueOAuthState(secret, stateUserID)
	require.NoError(t, err)

	h := &HTTPHandlers{
		Config: Config{
			ClientID:     "client-id",
			ClientSecret: "secret",
			PublicURL:    "https://admin.example",
			StateSecret:  secret,
		},
		Store: &callbackTestStore{},
		ActorUserID: func(*http.Request) (uuid.UUID, bool) {
			return uuid.New(), true
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/integrations/google-sheets/callback?code=auth-code&state="+state,
		http.NoBody,
	)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "oauth state does not match session")
}

func TestHTTPHandlers_deleteConnection_requiresActor(t *testing.T) {
	t.Parallel()
	h := &HTTPHandlers{
		Config: Config{ClientID: "client-id", ClientSecret: "secret"},
		Store:  &Store{},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/google-sheets", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
