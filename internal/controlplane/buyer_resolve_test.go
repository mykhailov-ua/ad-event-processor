package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ctrlhttp "ad-event-processor/internal/control/http"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResolveCampaignsCustomerID_BuyerUsesSessionCustomer(t *testing.T) {
	t.Parallel()
	h := &Handler{}
	custID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/campaigns", http.NoBody)
	ctx := authz.WithAuthenticatedUser(req.Context(), authz.AuthenticatedUser{
		UserID:     uuid.New(),
		Role:       ctrlhttp.RoleBuyer,
		CustomerID: custID,
	})
	req = req.WithContext(ctx)

	got, err := h.resolveCampaignsCustomerID(req, nil)
	require.NoError(t, err)
	require.Equal(t, custID, got)
}

func TestAuthenticatedUser_HasBoundCustomer(t *testing.T) {
	t.Parallel()
	require.True(t, (authz.AuthenticatedUser{Role: ctrlhttp.RoleUser}).HasBoundCustomer())
	require.True(t, (authz.AuthenticatedUser{Role: ctrlhttp.RoleBuyer}).HasBoundCustomer())
	require.True(t, (authz.AuthenticatedUser{Role: ctrlhttp.RoleMediaBuyer}).HasBoundCustomer())
	require.True(t, (authz.AuthenticatedUser{Role: ctrlhttp.RolePublisher}).HasBoundCustomer())
	require.False(t, (authz.AuthenticatedUser{Role: ctrlhttp.RoleAdmin}).HasBoundCustomer())
}
