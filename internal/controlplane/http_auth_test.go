package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/identity"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAuthAPI struct {
	loginFunc   func(ctx context.Context, email, password string, durationHours int32) (identity.LoginResult, error)
	revokeFunc  func(ctx context.Context, refreshToken string) error
	refreshFunc func(ctx context.Context, refreshToken string) (identity.RefreshResult, error)
}

func (m *mockAuthAPI) VerifyAPIKey(context.Context, string) (identity.AuthUser, error) {
	return identity.AuthUser{}, errors.New("unexpected call to VerifyAPIKey")
}

func (m *mockAuthAPI) VerifyToken(context.Context, string) (identity.AuthUser, error) {
	return identity.AuthUser{}, errors.New("unexpected call to VerifyToken")
}

func (m *mockAuthAPI) CreateAPIKey(context.Context, string, string) (identity.CreateAPIKeyResult, error) {
	return identity.CreateAPIKeyResult{}, errors.New("unexpected call to CreateAPIKey")
}

func (m *mockAuthAPI) Login(ctx context.Context, email, password string, durationHours int32) (identity.LoginResult, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password, durationHours)
	}
	return identity.LoginResult{}, errors.New("unexpected call to Login")
}

func (m *mockAuthAPI) Register(context.Context, string, string, string, string, string) (identity.RegisterResult, error) {
	return identity.RegisterResult{}, errors.New("unexpected call to Register")
}

func (m *mockAuthAPI) RevokeToken(ctx context.Context, refreshToken string) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, refreshToken)
	}
	return errors.New("unexpected call to RevokeToken")
}

func (m *mockAuthAPI) RefreshToken(ctx context.Context, refreshToken string) (identity.RefreshResult, error) {
	if m.refreshFunc != nil {
		return m.refreshFunc(ctx, refreshToken)
	}
	return identity.RefreshResult{}, errors.New("unexpected call to RefreshToken")
}

func TestAuthHandler_Login(t *testing.T) {
	cfg := &config.Config{
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	tokenMaker, err := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	require.NoError(t, err)

	mockClient := &mockAuthAPI{
		loginFunc: func(ctx context.Context, email, password string, durationHours int32) (identity.LoginResult, error) {
			if email == "test@example.com" && password == "correctPass" {
				return identity.LoginResult{
					AccessToken:  "access-token-jwt",
					RefreshToken: "refresh-token-uuid",
					User: identity.AuthUser{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000123"),
						Email:      "test@example.com",
						Role:       "admin",
						CustomerID: uuid.MustParse("00000000-0000-0000-0000-000000000456"),
					},
				}, nil
			}
			return identity.LoginResult{}, errors.New("invalid credentials")
		},
	}

	h := NewAuthHandler(NewAuthClientFromAPI(mockClient), tokenMaker, nil, cfg, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("ValidLogin", func(t *testing.T) {
		reqBody := map[string]string{"email": "test@example.com", "password": "correctPass"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()

		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		cookies := resp.Header().Values("Set-Cookie")
		require.Len(t, cookies, 3)

		var accessSet, refreshSet, csrfSet bool
		for _, c := range cookies {
			if strings.HasPrefix(c, "accessToken=") {
				accessSet = true
				assert.Contains(t, c, "HttpOnly")
				assert.NotContains(t, c, "Secure")
				assert.Contains(t, c, "SameSite=Strict")
				assert.Contains(t, c, "Max-Age=3600")
			}
			if strings.HasPrefix(c, "refreshToken=") {
				refreshSet = true
				assert.Contains(t, c, "HttpOnly")
				assert.NotContains(t, c, "Secure")
				assert.Contains(t, c, "SameSite=Strict")
				assert.Contains(t, c, "Max-Age=2592000")
			}
			if strings.HasPrefix(c, "csrfToken=") {
				csrfSet = true
				assert.NotContains(t, c, "HttpOnly")
				assert.NotContains(t, c, "Secure")
				assert.Contains(t, c, "SameSite=Strict")
				assert.Contains(t, c, "Max-Age=3600")
			}
		}
		assert.True(t, accessSet)
		assert.True(t, refreshSet)
		assert.True(t, csrfSet)
		assert.NotEmpty(t, resp.Header().Get("X-CSRF-Token"))

		var res map[string]UserDTO
		err := json.NewDecoder(resp.Body).Decode(&res)
		require.NoError(t, err)
		assert.Equal(t, "00000000-0000-0000-0000-000000000123", res["user"].ID)
		assert.Equal(t, RoleAdmin, res["user"].Role)
		assert.Contains(t, res["user"].Permissions, "customers:write")
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		reqBody := map[string]string{"email": "test@example.com", "password": "wrongPass"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		resp := httptest.NewRecorder()

		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	cfg := &config.Config{
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	tokenMaker, _ := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))

	var revokedToken string
	mockClient := &mockAuthAPI{
		revokeFunc: func(ctx context.Context, refreshToken string) error {
			revokedToken = refreshToken
			return nil
		},
	}

	h := NewAuthHandler(NewAuthClientFromAPI(mockClient), tokenMaker, nil, cfg, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "refreshToken", Value: "token-to-revoke"})
	resp := httptest.NewRecorder()

	mux.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.Equal(t, "token-to-revoke", revokedToken)

	cookies := resp.Header().Values("Set-Cookie")
	for _, c := range cookies {
		assert.Contains(t, c, "Max-Age=0")
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	cfg := &config.Config{
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	tokenMaker, _ := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))

	mockClient := &mockAuthAPI{
		refreshFunc: func(ctx context.Context, refreshToken string) (identity.RefreshResult, error) {
			if refreshToken == "valid-refresh" {
				return identity.RefreshResult{
					AccessToken:  "new-access",
					RefreshToken: "new-refresh",
				}, nil
			}
			return identity.RefreshResult{}, errors.New("invalid refresh")
		},
	}

	h := NewAuthHandler(NewAuthClientFromAPI(mockClient), tokenMaker, nil, cfg, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("ValidRefresh", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", http.NoBody)
		req.AddCookie(&http.Cookie{Name: "refreshToken", Value: "valid-refresh"})
		resp := httptest.NewRecorder()

		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		cookies := resp.Header().Values("Set-Cookie")
		require.Len(t, cookies, 3)
		assert.NotEmpty(t, resp.Header().Get("X-CSRF-Token"))
	})
}

func TestAuthHandler_Me(t *testing.T) {
	cfg := &config.Config{
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	tokenMaker, _ := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))

	userID := uuid.New()
	customerID := uuid.New()
	sessionID := uuid.New()
	token, err := tokenMaker.CreateToken(userID, sessionID, "admin", customerID, time.Hour)
	require.NoError(t, err)

	h := NewAuthHandler(nil, tokenMaker, nil, cfg, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req, _ := http.NewRequest("GET", "/api/v1/auth/me", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	resp := httptest.NewRecorder()

	mux.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var dto UserDTO
	err = json.NewDecoder(resp.Body).Decode(&dto)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), dto.ID)
	assert.Equal(t, RoleAdmin, dto.Role)
	assert.Equal(t, customerID.String(), dto.CustomerID)
	assert.Contains(t, dto.Permissions, "campaigns:write")
	assert.NotEmpty(t, resp.Header().Get("X-CSRF-Token"))
}

func TestAuthHandler_MeRedisOutage(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()
	_ = rdb.Close()

	cfg := &config.Config{
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	tokenMaker, err := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	require.NoError(t, err)

	userID := uuid.New()
	customerID := uuid.New()
	token, err := tokenMaker.CreateToken(userID, uuid.New(), RoleAdmin, customerID, time.Hour)
	require.NoError(t, err)

	h := NewAuthHandler(nil, tokenMaker, []redis.UniversalClient{rdb}, cfg, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req, _ := http.NewRequest("GET", "/api/v1/auth/me", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	resp := httptest.NewRecorder()

	mux.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "security check failed")
}
