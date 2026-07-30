package vendorserver

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"espx/internal/billing/db"
	"espx/internal/licensing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const revokeReasonManyFingerprints = "same_key_many_fingerprints"

type Server struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	privKey ed25519.PrivateKey
	now     func() time.Time
}

func New(pool *pgxpool.Pool, privKey ed25519.PrivateKey) *Server {
	return &Server{
		pool:    pool,
		queries: db.New(pool),
		privKey: privKey,
		now:     time.Now,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/licenses/issue", s.handleIssue)
	mux.HandleFunc("/v1/licenses/renew", s.handleRenew)
	mux.HandleFunc("/v1/licenses/revoke", s.handleRevoke)
	mux.HandleFunc("/v1/activate", s.handleActivate)
	mux.HandleFunc("/v1/heartbeat", s.handleHeartbeat)
}

type IssueRequest struct {
	LicenseKey      string               `json:"license_key"`
	CustomerName    string               `json:"customer_name"`
	Plan            string               `json:"plan"`
	ValidFrom       time.Time            `json:"valid_from"`
	ValidUntil      time.Time            `json:"valid_until"`
	GraceDays       int                  `json:"grace_days"`
	MaxActivations  int                  `json:"max_activations"`
	Limits          licensing.Limits     `json:"limits"`
	Features        licensing.FeatureSet `json:"features"`
	SupportTier     string               `json:"support_tier"`
}

func (s *Server) handleIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req IssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limitsJSON, _ := json.Marshal(req.Limits)
	featuresJSON, _ := json.Marshal(req.Features)

	if req.ValidFrom.IsZero() {
		req.ValidFrom = s.now()
	}
	maxActivations := int32(req.MaxActivations)
	if maxActivations <= 0 {
		maxActivations = 1
	}

	_, err := s.queries.InsertVendorLicense(r.Context(), db.InsertVendorLicenseParams{
		LicenseKey:      req.LicenseKey,
		CustomerName:    req.CustomerName,
		PlanCode:        req.Plan,
		ValidFrom:       pgtype.Timestamptz{Time: req.ValidFrom, Valid: true},
		ValidUntil:      pgtype.Timestamptz{Time: req.ValidUntil, Valid: true},
		GraceDays:       int32(req.GraceDays),
		LimitsJson:      limitsJSON,
		FeaturesJson:    featuresJSON,
		SupportTier:     req.SupportTier,
		Revoked:         false,
		MaxActivations:  maxActivations,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "issued"})
}

type RenewRequest struct {
	LicenseKey string    `json:"license_key"`
	ValidUntil time.Time `json:"valid_until"`
}

func (s *Server) handleRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RenewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := s.queries.RenewVendorLicense(r.Context(), db.RenewVendorLicenseParams{
		LicenseKey: req.LicenseKey,
		ValidUntil: pgtype.Timestamptz{Time: req.ValidUntil, Valid: true},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = s.queries.RecordVendorRenewalEvent(r.Context(), db.RecordVendorRenewalEventParams{
		LicenseKey:    req.LicenseKey,
		NewValidUntil: pgtype.Timestamptz{Time: req.ValidUntil, Valid: true},
	})

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "renewed"})
}

type RevokeRequest struct {
	LicenseKey string `json:"license_key"`
}

func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.queries.RevokeVendorLicense(r.Context(), req.LicenseKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}

type ActivateRequest struct {
	LicenseKey   string `json:"license_key"`
	DeploymentID string `json:"deployment_id"`
	Fingerprint  string `json:"fingerprint"`
}

func (s *Server) handleActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, status, err := s.activateOrHeartbeat(r.Context(), req.LicenseKey, req.DeploymentID, req.Fingerprint, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if status != http.StatusOK {
		http.Error(w, statusText(status), status)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}

type HeartbeatRequest struct {
	LicenseKey    string `json:"license_key"`
	DeploymentID  string `json:"deployment_id"`
	Fingerprint   string `json:"fingerprint"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, status, err := s.activateOrHeartbeat(r.Context(), req.LicenseKey, req.DeploymentID, req.Fingerprint, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if status != http.StatusOK {
		http.Error(w, statusText(status), status)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func statusText(code int) string {
	switch code {
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not found"
	case http.StatusBadRequest:
		return "bad request"
	default:
		return http.StatusText(code)
	}
}

func (s *Server) activateOrHeartbeat(ctx context.Context, licenseKey, deploymentID, fingerprint string, isActivate bool) (string, int, error) {
	lic, err := s.queries.GetVendorLicense(ctx, licenseKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", http.StatusNotFound, nil
		}
		return "", 0, err
	}
	if lic.Revoked {
		slog.Warn("license activation denied", "license_key", licenseKey, "reason", "revoked")
		return "", http.StatusForbidden, nil
	}

	depID, err := uuid.Parse(deploymentID)
	if err != nil {
		return "", http.StatusBadRequest, nil
	}

	acts, err := s.listActivations(ctx, licenseKey)
	if err != nil {
		return "", 0, err
	}

	var deployment *licensing.DeploymentRecord
	depRow, err := s.queries.GetVendorDeployment(ctx, pgtype.UUID{Bytes: depID, Valid: true})
	if err == nil {
		deployment = &licensing.DeploymentRecord{
			DeploymentID: uuid.UUID(depRow.DeploymentID.Bytes).String(),
			LicenseKey:   depRow.LicenseKey,
			Fingerprint:  depRow.Fingerprint,
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", 0, err
	}

	var decision licensing.ActivationDecision
	if isActivate {
		decision = licensing.EvaluateActivate(fingerprint, licenseKey, lic.MaxActivations, acts, deployment)
	} else {
		decision = licensing.EvaluateHeartbeat(fingerprint, licenseKey, acts, deployment)
	}
	if !decision.Allow {
		slog.Warn("license activation denied",
			"license_key", licenseKey,
			"deployment_id", deploymentID,
			"fingerprint", fingerprint,
			"activate", isActivate,
			"reason", decision.DenyReason,
		)
		if decision.DenyReason == licensing.ErrDeploymentNotFound.Error() {
			return "", http.StatusNotFound, nil
		}
		return "", http.StatusForbidden, nil
	}

	if decision.BindActivation {
		_, err = s.queries.UpsertLicenseActivation(ctx, db.UpsertLicenseActivationParams{
			LicenseKey:   licenseKey,
			Fingerprint:  fingerprint,
			DeploymentID: pgtype.UUID{Bytes: depID, Valid: true},
		})
		if err != nil {
			return "", 0, err
		}
	}

	_, err = s.queries.UpsertVendorDeployment(ctx, db.UpsertVendorDeploymentParams{
		DeploymentID: pgtype.UUID{Bytes: depID, Valid: true},
		LicenseKey:   licenseKey,
		Fingerprint:  fingerprint,
	})
	if err != nil {
		return "", 0, err
	}

	if decision.FlagManyFingerprints {
		detail, _ := json.Marshal(map[string]string{
			"license_key":   licenseKey,
			"deployment_id": deploymentID,
			"fingerprint":   fingerprint,
		})
		_, _ = s.queries.InsertLicenseRevokeFlag(ctx, db.InsertLicenseRevokeFlagParams{
			LicenseKey: licenseKey,
			Reason:     revokeReasonManyFingerprints,
			DetailJson: detail,
		})
		slog.Warn("license anomaly flagged",
			"license_key", licenseKey,
			"reason", revokeReasonManyFingerprints,
		)
	}

	token, err := s.signToken(lic, depID, fingerprint)
	if err != nil {
		return "", 0, err
	}
	return token, http.StatusOK, nil
}

func (s *Server) listActivations(ctx context.Context, licenseKey string) ([]licensing.ActivationRecord, error) {
	rows, err := s.queries.ListLicenseActivationsByKey(ctx, licenseKey)
	if err != nil {
		return nil, err
	}
	out := make([]licensing.ActivationRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, licensing.ActivationRecord{
			LicenseKey:   row.LicenseKey,
			Fingerprint:  row.Fingerprint,
			DeploymentID: uuid.UUID(row.DeploymentID.Bytes).String(),
		})
	}
	return out, nil
}

func (s *Server) signToken(lic db.VendorLicense, depID uuid.UUID, fingerprint string) (string, error) {
	var limits licensing.Limits
	_ = json.Unmarshal(lic.LimitsJson, &limits)

	var features licensing.FeatureSet
	_ = json.Unmarshal(lic.FeaturesJson, &features)

	now := s.now().UTC()
	validUntil := licensing.CapHeartbeatValidUntil(lic.ValidUntil.Time, now)

	claims := licensing.LicenseClaims{
		Issuer:       "espx-license",
		Subject:      uuid.NewString(),
		DeploymentID: depID.String(),
		CustomerName: lic.CustomerName,
		Plan:         lic.PlanCode,
		ValidFrom:    lic.ValidFrom.Time.UTC(),
		ValidUntil:   validUntil,
		GraceDays:    int(lic.GraceDays),
		Limits:       limits,
		Features:     features,
		SupportTier:  lic.SupportTier,
	}
	claims.Bind.Mode = "fingerprint"
	claims.Bind.Fingerprint = fingerprint

	return licensing.SignJWT(claims, s.privKey, licensing.DefaultLicenseKeyID)
}
