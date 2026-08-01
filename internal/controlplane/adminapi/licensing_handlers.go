package adminapi

import (
	"errors"
	"net/http"
	"time"

	"espx/internal/ledger/db"
	"espx/pkg/httpresponse"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LicenseStatusResponse struct {
	DeploymentID string `json:"deployment_id"`
	State        string `json:"state"`
	ValidUntil   string `json:"valid_until"`
}

type LicensingHTTPHandlers struct {
	Pool              *pgxpool.Pool
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (licensing *LicensingHTTPHandlers) Register(mux *http.ServeMux) {
	if licensing == nil || licensing.Pool == nil {
		return
	}
	limit := licensing.ApplyRateLimit
	perm := licensing.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/license/status", limit(perm("customers:read", licensing.getLicenseStatus)))
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
