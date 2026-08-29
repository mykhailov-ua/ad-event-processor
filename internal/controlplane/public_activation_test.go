package controlplane

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/platformadmin"

	"github.com/stretchr/testify/require"
)

type activationStub struct{}

func (activationStub) ActivateOwner(_ context.Context, _ platformadmin.ActivateOwnerRequest) (platformadmin.ActivatedOwner, error) {
	return platformadmin.ActivatedOwner{}, nil
}

func (activationStub) AcceptTeamInvite(_ context.Context, _ platformadmin.AcceptTeamInviteRequest) (platformadmin.ActivatedOwner, error) {
	return platformadmin.ActivatedOwner{}, nil
}

func TestPublicActivate_rejectsEmptyBody(t *testing.T) {
	h := &platformadmin.PublicHTTPHandlers{
		Activation: activationStub{},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/activate", bytes.NewReader(nil))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestNormalizeTeamRole_allowsBuyer(t *testing.T) {
	svc := &Service{}
	role, err := svc.NormalizeTeamRole("B")
	require.NoError(t, err)
	require.Equal(t, "B", role)
}
