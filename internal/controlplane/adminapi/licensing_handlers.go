package adminapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"espx/internal/ledger/db"
	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LicenseStatusResponse struct {
	DeploymentID string `json:"deployment_id"`
	State        string `json:"state"`
	ValidUntil   string `json:"valid_until"`
}

type ApplyLicenseRequest struct {
	Token string `json:"token"`
}

type LicenseService interface {
	ApplyLicenseToken(ctx context.Context, token string) error
}

type LicensingHTTPHandlers struct {
	Pool                  *pgxpool.Pool
	LicenseService        LicenseService
	ApplyRateLimit        func(http.HandlerFunc) http.HandlerFunc
	LicenseApplyRateLimit func(http.HandlerFunc) http.HandlerFunc
	RequirePermission     func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError     func(http.ResponseWriter, error)
}

func (licensing *LicensingHTTPHandlers) Register(mux *http.ServeMux) {
	if licensing == nil || licensing.Pool == nil {
		return
	}
	limit := licensing.ApplyRateLimit
	applyLimit := licensing.LicenseApplyRateLimit
	perm := licensing.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if applyLimit == nil {
		applyLimit = limit
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/license/status", limit(perm("customers:read", licensing.getLicenseStatus)))
	if licensing.LicenseService != nil {
		mux.HandleFunc("POST /api/v1/license/apply", limit(applyLimit(perm("settings:write", licensing.postLicenseApply))))
	}
}

func (licensing *LicensingHTTPHandlers) getLicenseStatus(w http.ResponseWriter, r *http.Request) {
	q := db.New(licensing.Pool)
	licRow, err := q.GetLicenseStatus(r.Context())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "license not configured")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	resp := LicenseStatusResponse{
		DeploymentID: licRow.DeploymentID.String(),
		State:        licRow.State,
	}
	if licRow.ValidUntil.Valid {
		resp.ValidUntil = licRow.ValidUntil.Time.Format(time.RFC3339)
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (licensing *LicensingHTTPHandlers) postLicenseApply(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[ApplyLicenseRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := licensing.LicenseService.ApplyLicenseToken(r.Context(), req.Token); err != nil {
		if licensing.WriteServiceError != nil {
			licensing.WriteServiceError(w, err)
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	q := db.New(licensing.Pool)
	licRow, err := q.GetLicenseStatus(r.Context())
	if err != nil {
		httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "applied"})
		return
	}
	resp := LicenseStatusResponse{
		DeploymentID: licRow.DeploymentID.String(),
		State:        licRow.State,
	}
	if licRow.ValidUntil.Valid {
		resp.ValidUntil = licRow.ValidUntil.Time.Format(time.RFC3339)
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}
