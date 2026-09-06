package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestPostbackConfig_forbiddenForeignCampaign(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: postback RBAC scope before DB write")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ownedCustomer := uuid.New()
	foreignCustomer := uuid.New()
	ownedCampaign := uuid.New()
	foreignCampaign := uuid.New()

	for _, row := range []struct {
		customerID uuid.UUID
		campaignID uuid.UUID
		name       string
	}{
		{ownedCustomer, ownedCampaign, "owned"},
		{foreignCustomer, foreignCampaign, "foreign"},
	} {
		_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 0, 'USD')`, row.customerID, row.name)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, $2, 'ACTIVE', $3)`, row.campaignID, row.name, row.customerID)
		require.NoError(t, err)
	}

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	authMW, tokenMaker := integrationTestAuth(t, redisClient, cfg)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, err := json.Marshal(campaign.UpdatePostbackConfigRequest{
		Provider:    "webhook",
		URLTemplate: "https://example.com/postback?click_id={click_id}",
		TargetEvent: "conversion",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/postbacks/config/"+foreignCampaign.String(), bytes.NewReader(body))
	withSessionUser(req, tokenMaker, ctrlhttp.RoleUser, ownedCustomer)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}
