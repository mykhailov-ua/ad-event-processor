package controlplane

import (
	"context"
	"errors"
	"net/http"
	"time"

	"ad-event-processor/internal/ledger/db"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const licenseStateUnconfigured = "UNCONFIGURED"

type LicenseStatusResponse struct {
	DeploymentID    string `json:"deployment_id"`
	State           string `json:"state"`
	ValidUntil      string `json:"valid_until,omitempty"`
	HostFingerprint string `json:"host_fingerprint,omitempty"`
	HWIDv2          string `json:"hwid_v2,omitempty"`
	HWIDMatch       *bool  `json:"hwid_match,omitempty"`
	DaysToExpiry    int    `json:"days_to_expiry,omitempty"`
}

type ApplyLicenseRequest struct {
	Token string `json:"token"`
}

type LicenseService interface {
	ApplyLicenseToken(ctx context.Context, token string) error
}

type LicenseDiagnosticsProvider func() (licensing.LicenseDiagnostics, bool)

type LicensingHTTPHandlers struct {
	Pool                  *pgxpool.Pool
	LicenseService        LicenseService
	LicenseDiagnostics    LicenseDiagnosticsProvider
	ApplyRateLimit        func(http.HandlerFunc) http.HandlerFunc
	LicenseApplyRateLimit func(http.HandlerFunc) http.HandlerFunc
	RequirePermission     func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError     func(http.ResponseWriter, error)
}

func (h *LicensingHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Pool == nil {
		return
	}
	limit := h.ApplyRateLimit
	applyLimit := h.LicenseApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if applyLimit == nil {
		applyLimit = limit
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/license/status", limit(perm("customers:read", h.getLicenseStatus)))
	if h.LicenseService != nil {
		mux.HandleFunc("POST /api/v1/license/apply", limit(applyLimit(perm("settings:write", h.postLicenseApply))))
	}
}

func (h *LicensingHTTPHandlers) getLicenseStatus(w http.ResponseWriter, r *http.Request) {
	q := db.New(h.Pool)
	licRow, err := q.GetLicenseStatus(r.Context())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.JSON(w, http.StatusOK, toLicenseStatusResponse("", licenseStateUnconfigured, time.Time{}, false, h.LicenseDiagnostics))
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	resp := toLicenseStatusResponse(
		licRow.DeploymentID.String(),
		licRow.State,
		licRow.ValidUntil.Time,
		licRow.ValidUntil.Valid,
		h.LicenseDiagnostics,
	)
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *LicensingHTTPHandlers) postLicenseApply(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[ApplyLicenseRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.LicenseService.ApplyLicenseToken(r.Context(), req.Token); err != nil {
		if h.WriteServiceError != nil {
			h.WriteServiceError(w, err)
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	q := db.New(h.Pool)
	licRow, err := q.GetLicenseStatus(r.Context())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.JSON(w, http.StatusOK, toLicenseStatusResponse("", licenseStateUnconfigured, time.Time{}, false, h.LicenseDiagnostics))
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	resp := toLicenseStatusResponse(
		licRow.DeploymentID.String(),
		licRow.State,
		licRow.ValidUntil.Time,
		licRow.ValidUntil.Valid,
		h.LicenseDiagnostics,
	)
	httpresponse.JSON(w, http.StatusOK, resp)
}

func toLicenseStatusResponse(deploymentID, state string, validUntil time.Time, validUntilSet bool, diagFn LicenseDiagnosticsProvider) LicenseStatusResponse {
	resp := LicenseStatusResponse{
		DeploymentID: deploymentID,
		State:        state,
	}
	if validUntilSet && !validUntil.IsZero() {
		resp.ValidUntil = validUntil.UTC().Format(time.RFC3339)
	}

	var diag licensing.LicenseDiagnostics
	diagOK := false
	if diagFn != nil {
		diag, diagOK = diagFn()
	}
	if diagOK {
		if resp.DeploymentID == "" && diag.DeploymentID != "" {
			resp.DeploymentID = diag.DeploymentID
		}
		resp.HostFingerprint = diag.HostFingerprint
		resp.HWIDv2 = diag.HostHWID
		if diag.DaysToExpiry > 0 {
			resp.DaysToExpiry = diag.DaysToExpiry
		}
		if licensing.BindModeHard(diag.BindMode) && (diag.BindHWIDHash != "" || diag.BindFingerprint != "") {
			match := diag.HWIDMatch
			resp.HWIDMatch = &match
		}
		return resp
	}

	resp.HostFingerprint = licensing.HostFingerprint()
	resp.HWIDv2 = licensing.HostHWID()
	return resp
}
