package campaign

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCloneCampaignHTTP_missingIdempotencyKey_holdout(t *testing.T) {
	t.Parallel()

	h := &CampaignsHTTPHandlers{Campaigns: listAuxRouteCampaignStub{}}
	mux := http.NewServeMux()
	h.Register(mux)

	campaignID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/campaigns/"+campaignID.String()+"/clone",
		strings.NewReader(`{"name_suffix":" (copy)"}`),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Idempotency-Key")
}
