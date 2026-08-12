package adminapi

import (
	"context"
	"net/http"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/doctor"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"
)

type PlatformConfigReader func(ctx context.Context) (platformconfig.Config, error)

type DoctorHTTPHandlers struct {
	Config            *config.Config
	PlatformConfig    PlatformConfigReader
	ProbeDeps         doctor.ProbeDeps
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError func(http.ResponseWriter, error)
}

type DoctorResponseDTO struct {
	Overall          string                  `json:"overall"`
	Checks           []doctor.DoctorCheckDTO `json:"checks"`
	ClickURLTemplate string                  `json:"click_url_template"`
	TrackingDomain   string                  `json:"tracking_domain"`
	RtbMode          string                  `json:"rtb_mode,omitempty"`
	RtbEnabled       bool                    `json:"rtb_enabled"`
}

func (h *DoctorHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.PlatformConfig == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/ops/doctor", limit(perm("shards:read", h.getDoctor)))
}

func (h *DoctorHTTPHandlers) getDoctor(w http.ResponseWriter, r *http.Request) {
	platCfg, err := h.PlatformConfig(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	deps := h.ProbeDeps
	if deps.Config == nil {
		deps.Config = h.Config
	}

	report := doctor.RunPlatform(r.Context(), deps, platCfg, doctor.Options{
		Timeout: 30 * time.Second,
	})
	checks := doctor.ReportToDTO(report)

	rtbMode := ""
	rtbEnabled := false
	if h.Config != nil {
		rtbMode = h.Config.RtbMode
		rtbEnabled = h.Config.RtbEnabled()
	}

	httpresponse.JSON(w, http.StatusOK, DoctorResponseDTO{
		Overall:          doctor.OverallStatus(checks),
		Checks:           checks,
		ClickURLTemplate: platformconfig.ClickURLTemplate(platCfg.TrackingDomain),
		TrackingDomain:   platCfg.TrackingDomain,
		RtbMode:          rtbMode,
		RtbEnabled:       rtbEnabled,
	})
}

func (h *DoctorHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}
