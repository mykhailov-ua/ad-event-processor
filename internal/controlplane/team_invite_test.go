package controlplane_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/controlplane"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type teamGovStub struct {
	invited bool
}

func (s *teamGovStub) InviteTeamMember(_ context.Context, _ uuid.UUID, _, _ string) (controlplane.TeamMemberDTO, error) {
	s.invited = true
	return controlplane.TeamMemberDTO{}, nil
}

func (s *teamGovStub) UpdateTeamMember(_ context.Context, _, _ uuid.UUID, _ controlplane.UpdateTeamMemberRequest) (controlplane.TeamMemberDTO, error) {
	return controlplane.TeamMemberDTO{}, nil
}

func (s *teamGovStub) ListTeamBudgetApprovals(_ context.Context, _ uuid.UUID) ([]controlplane.TeamBudgetApprovalDTO, error) {
	return nil, nil
}

func (s *teamGovStub) ResolveTeamBudgetApproval(_ context.Context, _, _, _ uuid.UUID, _ bool) error {
	return nil
}

func TestTeamInvite_mediaBuyerForbidden(t *testing.T) {
	stub := &teamGovStub{}
	customerID := uuid.New()
	h := &controlplane.TeamHTTPHandlers{
		Team:       &controlplane.TeamOverviewService{},
		Governance: stub,
		RequireTeamWrite: func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
			}
		},
		ResolveCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return customerID, nil
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body, err := json.Marshal(controlplane.InviteTeamMemberRequest{Email: "buyer@test.com", Role: "MB"})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/members", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.False(t, stub.invited)
}
