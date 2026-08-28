package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *CampaignsHTTPHandlers) registerCampaignWizardRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	write := []string{"campaigns:write"}
	read := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/campaigns/onboarding-templates", limit(perm(read, h.listOnboardingTemplates)))
	mux.HandleFunc("GET /api/v1/campaigns/wizard/session", limit(perm(write, h.getCampaignWizardSession)))
	mux.HandleFunc("POST /api/v1/campaigns/wizard/session", limit(perm(write, h.postCampaignWizardSession)))
}

type campaignWizardService interface {
	CreateCampaignWizardSession(ctx context.Context, customerID uuid.UUID, templateKey string) (CampaignWizardSessionDTO, error)
	GetCampaignWizardSession(ctx context.Context, sessionID uuid.UUID) (CampaignWizardSessionDTO, error)
	UpdateCampaignWizardSessionStep(ctx context.Context, sessionID uuid.UUID, step string, payload json.RawMessage) (CampaignWizardSessionDTO, error)
	CommitCampaignWizardSession(ctx context.Context, sessionID uuid.UUID, idempotencyKey string, publish bool) (CampaignWizardCommitResult, error)
}

type CampaignWizardSessionRequest struct {
	Action         string          `json:"action"`
	SessionID      string          `json:"session_id,omitempty"`
	CustomerID     string          `json:"customer_id,omitempty"`
	TemplateKey    string          `json:"template_key,omitempty"`
	Step           string          `json:"step,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	Publish        *bool           `json:"publish,omitempty"`
}

func (h *CampaignsHTTPHandlers) listOnboardingTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := ListOnboardingTemplates()
	if err != nil {
		if h.WriteServiceError != nil {
			h.WriteServiceError(w, err)
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, templates)
}

func (h *CampaignsHTTPHandlers) getCampaignWizardSession(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseCampaignWizardSessionID(w, r)
	if !ok {
		return
	}
	svc, ok := h.Campaigns.(campaignWizardService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "wizard not configured")
		return
	}
	dto, err := svc.GetCampaignWizardSession(r.Context(), sessionID)
	if err != nil {
		writeCampaignWizardError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *CampaignsHTTPHandlers) postCampaignWizardSession(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[CampaignWizardSessionRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	svc, ok := h.Campaigns.(campaignWizardService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "wizard not configured")
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "create":
		h.handleCampaignWizardCreate(w, r, svc, req)
	case "update":
		h.handleCampaignWizardUpdate(w, r, svc, req)
	case "commit":
		h.handleCampaignWizardCommit(w, r, svc, req)
	default:
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "action must be create, update, or commit")
	}
}

func (h *CampaignsHTTPHandlers) handleCampaignWizardCreate(
	w http.ResponseWriter,
	r *http.Request,
	svc campaignWizardService,
	req CampaignWizardSessionRequest,
) {
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil || customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id is required")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err = h.ResolveCustomerID(r, &customerID)
		if err != nil {
			if h.WriteServiceError != nil {
				h.WriteServiceError(w, err)
				return
			}
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
	}
	dto, err := svc.CreateCampaignWizardSession(r.Context(), customerID, req.TemplateKey)
	if err != nil {
		writeCampaignWizardError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, dto)
}

func (h *CampaignsHTTPHandlers) handleCampaignWizardUpdate(
	w http.ResponseWriter,
	r *http.Request,
	svc campaignWizardService,
	req CampaignWizardSessionRequest,
) {
	sessionID, err := uuid.Parse(strings.TrimSpace(req.SessionID))
	if err != nil || sessionID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "session_id is required")
		return
	}
	step := strings.TrimSpace(req.Step)
	if step == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "step is required")
		return
	}
	if len(req.Payload) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload is required")
		return
	}
	dto, err := svc.UpdateCampaignWizardSessionStep(r.Context(), sessionID, step, req.Payload)
	if err != nil {
		writeCampaignWizardError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *CampaignsHTTPHandlers) handleCampaignWizardCommit(
	w http.ResponseWriter,
	r *http.Request,
	svc campaignWizardService,
	req CampaignWizardSessionRequest,
) {
	sessionID, err := uuid.Parse(strings.TrimSpace(req.SessionID))
	if err != nil || sessionID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "session_id is required")
		return
	}
	publish := false
	if req.Publish != nil {
		publish = *req.Publish
	}
	result, err := svc.CommitCampaignWizardSession(r.Context(), sessionID, req.IdempotencyKey, publish)
	if err != nil {
		writeCampaignWizardError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func parseCampaignWizardSessionID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if raw == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "session_id query parameter is required")
		return uuid.Nil, false
	}
	sessionID, err := uuid.Parse(raw)
	if err != nil || sessionID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid session_id")
		return uuid.Nil, false
	}
	return sessionID, true
}

func writeCampaignWizardError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrCampaignWizardSessionNotFound):
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrCampaignWizardSessionExpired):
		httpresponse.Error(w, http.StatusGone, "SESSION_EXPIRED", err.Error())
	case errors.Is(err, ErrCampaignWizardIncomplete):
		httpresponse.Error(w, http.StatusUnprocessableEntity, "WIZARD_INCOMPLETE", err.Error())
	default:
		if errors.Is(err, ErrValidation) {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		if errors.Is(err, ErrForbidden) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
	}
}
