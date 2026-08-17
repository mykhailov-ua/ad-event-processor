package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResolveCampaignsCustomerID_BuyerUsesSessionCustomer(t *testing.T) {
	t.Parallel()
	h := &Handler{}
	custID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/campaigns", http.NoBody)
	ctx := context.WithValue(req.Context(), UserContextKey, AuthenticatedUser{
		UserID:     uuid.New(),
		Role:       RoleBuyer,
		CustomerID: custID,
	})
	req = req.WithContext(ctx)

	got, err := h.resolveCampaignsCustomerID(req, nil)
	require.NoError(t, err)
	require.Equal(t, custID, got)
}

func TestAuthenticatedUser_HasBoundCustomer(t *testing.T) {
	t.Parallel()
	require.True(t, (AuthenticatedUser{Role: RoleUser}).HasBoundCustomer())
	require.True(t, (AuthenticatedUser{Role: RoleBuyer}).HasBoundCustomer())
	require.True(t, (AuthenticatedUser{Role: RoleMediaBuyer}).HasBoundCustomer())
	require.True(t, (AuthenticatedUser{Role: RolePublisher}).HasBoundCustomer())
	require.False(t, (AuthenticatedUser{Role: RoleAdmin}).HasBoundCustomer())
}
