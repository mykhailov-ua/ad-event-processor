package controlplane_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/controlplane"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type campaignFraudStub struct {
	cfg        controlplane.CampaignFraudConfigDTO
	getErr     error
	patch      controlplane.PatchCampaignFraudRequest
	patchOK    bool
	preview    controlplane.PreviewCampaignFraudRequest
	previewOK  bool
	previewOut controlplane.CampaignFraudPreviewDTO
}

func (s *campaignFraudStub) GetCampaignFraudConfig(_ context.Context, campaignID uuid.UUID) (controlplane.CampaignFraudConfigDTO, error) {
	if s.getErr != nil {
		return controlplane.CampaignFraudConfigDTO{}, s.getErr
	}
	out := s.cfg
	if out.CampaignID == "" {
		out.CampaignID = campaignID.String()
	}
	return out, nil
}

func (s *campaignFraudStub) UpdateCampaignFraudConfig(_ context.Context, campaignID uuid.UUID, req controlplane.PatchCampaignFraudRequest) (controlplane.CampaignFraudConfigDTO, error) {
	s.patch = req
	s.patchOK = true
	if s.getErr != nil {
		return controlplane.CampaignFraudConfigDTO{}, s.getErr
	}
	out := s.cfg
	out.CampaignID = campaignID.String()
	if req.Preset != nil && *req.Preset == "aggressive" {
		out.FraudThresholdPass = 20
		out.FraudThresholdSuspect = 45
		out.FraudThresholdIVT = 65
		out.FraudThresholdBlock = 85
	}
	if req.SilentRejectEnabled != nil {
		out.SilentRejectEnabled = *req.SilentRejectEnabled
	}
	return out, nil
}

func (s *campaignFraudStub) PreviewCampaignFraudImpact(_ context.Context, campaignID uuid.UUID, req controlplane.PreviewCampaignFraudRequest) (controlplane.CampaignFraudPreviewDTO, error) {
	s.preview = req
	s.previewOK = true
	if s.getErr != nil {
		return controlplane.CampaignFraudPreviewDTO{}, s.getErr
	}
	if s.previewOut.CampaignID != "" {
		return s.previewOut, nil
	}
	return controlplane.CampaignFraudPreviewDTO{
		CampaignID:    campaignID.String(),
		AffectedIPs7d: 12,
		SampleSize:    100,
		ByTier: controlplane.FraudPreviewTierCountsDTO{
			Suspect: 5,
			IVT:     4,
			Block:   3,
		},
		Disclaimer: "estimate",
	}, nil
}

func mapPublisherTestError(err error) (status int, code string, message string) {
	_ = err
	return http.StatusInternalServerError, "INTERNAL", "internal error"
}

func newCampaignFraudHandlers(stub *campaignFraudStub) *controlplane.CampaignsHTTPHandlers {
	return &controlplane.CampaignsHTTPHandlers{
		Campaigns:     &campaignListStub{},
		CampaignFraud: stub,
		RequireAnyPermission: func(required []string, next http.HandlerFunc) http.HandlerFunc {
			perms := map[string]bool{
				"campaigns:read":        true,
				"campaigns:read:masked": true,
				"campaigns:write":       true,
			}
			return func(w http.ResponseWriter, r *http.Request) {
				for _, p := range required {
					if perms[p] {
						next(w, r)
						return
					}
				}
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
			}
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			status, code, msg := mapPublisherTestError(err)
			httpresponse.Error(w, status, code, msg)
		},
	}
}

func TestGetCampaignFraud_returnsConfig(t *testing.T) {
	campaignID := uuid.New()
	stub := &campaignFraudStub{
		cfg: controlplane.CampaignFraudConfigDTO{
			CampaignID:            campaignID.String(),
			FraudThresholdPass:    30,
			FraudThresholdSuspect: 60,
			FraudThresholdIVT:     80,
			FraudThresholdBlock:   100,
			SilentRejectEnabled:   true,
		},
	}
	h := newCampaignFraudHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campaignID.String()+"/fraud", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var out controlplane.CampaignFraudConfigDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, uint8(30), out.FraudThresholdPass)
	require.True(t, out.SilentRejectEnabled)
}

func TestPatchCampaignFraud_legacySilentRejectJSONField(t *testing.T) {
	campaignID := uuid.New()
	stub := &campaignFraudStub{
		cfg: controlplane.CampaignFraudConfigDTO{
			FraudThresholdPass:    30,
			FraudThresholdSuspect: 60,
			FraudThresholdIVT:     80,
			FraudThresholdBlock:   100,
		},
	}
	h := newCampaignFraudHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	body := `{"ghost_ivt_enabled":true}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campaignID.String()+"/fraud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, stub.patchOK)
	require.NotNil(t, stub.patch.SilentRejectEnabled)
	require.True(t, *stub.patch.SilentRejectEnabled)

	var out controlplane.CampaignFraudConfigDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.True(t, out.SilentRejectEnabled)
}

func TestPatchCampaignFraud_appliesPreset(t *testing.T) {
	campaignID := uuid.New()
	stub := &campaignFraudStub{
		cfg: controlplane.CampaignFraudConfigDTO{
			FraudThresholdPass:    30,
			FraudThresholdSuspect: 60,
			FraudThresholdIVT:     80,
			FraudThresholdBlock:   100,
		},
	}
	h := newCampaignFraudHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	body := `{"preset":"aggressive","silent_reject_enabled":true}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campaignID.String()+"/fraud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, stub.patchOK)
	require.NotNil(t, stub.patch.Preset)
	require.Equal(t, "aggressive", *stub.patch.Preset)

	var out controlplane.CampaignFraudConfigDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, uint8(20), out.FraudThresholdPass)
	require.True(t, out.SilentRejectEnabled)
}

func TestPatchCampaignFraud_invalidJSON(t *testing.T) {
	campaignID := uuid.New()
	stub := &campaignFraudStub{}
	h := newCampaignFraudHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campaignID.String()+"/fraud", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPostCampaignFraudPreview_returnsEstimate(t *testing.T) {
	campaignID := uuid.New()
	stub := &campaignFraudStub{
		cfg: controlplane.CampaignFraudConfigDTO{
			FraudThresholdPass:    30,
			FraudThresholdSuspect: 60,
			FraudThresholdIVT:     80,
			FraudThresholdBlock:   100,
		},
	}
	h := newCampaignFraudHandlers(stub)
	mux := http.NewServeMux()
	h.Register(mux)

	body := `{"fraud_threshold_pass":25,"fraud_threshold_suspect":55}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+campaignID.String()+"/fraud/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, stub.previewOK)
	require.NotNil(t, stub.preview.FraudThresholdPass)
	require.Equal(t, uint8(25), *stub.preview.FraudThresholdPass)

	var out controlplane.CampaignFraudPreviewDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, int64(12), out.AffectedIPs7d)
	require.Equal(t, int64(100), out.SampleSize)
	require.Equal(t, int64(5), out.ByTier.Suspect)
}
