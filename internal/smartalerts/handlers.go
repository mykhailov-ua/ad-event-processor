package smartalerts

import (
	"context"
	"net/http"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type Service interface {
	ListSmartAlertRules(ctx context.Context, customerID uuid.UUID) ([]RuleDTO, error)
	CreateSmartAlertRule(ctx context.Context, req UpsertRuleRequest) (RuleDTO, error)
	UpdateSmartAlertRule(ctx context.Context, ruleID uuid.UUID, req UpsertRuleRequest) (RuleDTO, error)
	DeleteSmartAlertRule(ctx context.Context, ruleID uuid.UUID) error
	ListSmartAlertHistory(ctx context.Context, customerID uuid.UUID, limit int) ([]EventDTO, error)
	AckSmartAlertEvent(ctx context.Context, eventID, actorID uuid.UUID) error
}

type RuleDTO = SmartAlertRuleDTO
type EventDTO = SmartAlertEventDTO
type UpsertRuleRequest = UpsertSmartAlertRuleRequest

type HTTPHandlers struct {
	Service           Service
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	ResolveActorID    func(context.Context) uuid.UUID
}

type SmartAlertRuleDTO struct {
	ID            string    `json:"id"`
	CustomerID    string    `json:"customer_id"`
	CampaignID    string    `json:"campaign_id,omitempty"`
	Name          string    `json:"name"`
	Metric        string    `json:"metric"`
	Operator      string    `json:"operator"`
	Threshold     float64   `json:"threshold"`
	WindowMinutes int       `json:"window_minutes"`
	WebhookURL    string    `json:"webhook_url"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SmartAlertEventDTO struct {
	ID            string     `json:"id"`
	RuleID        string     `json:"rule_id"`
	CustomerID    string     `json:"customer_id"`
	CampaignID    string     `json:"campaign_id,omitempty"`
	WindowStart   time.Time  `json:"window_start"`
	WindowEnd     time.Time  `json:"window_end"`
	Metric        string     `json:"metric"`
	Operator      string     `json:"operator"`
	Threshold     float64    `json:"threshold"`
	ObservedValue float64    `json:"observed_value"`
	WebhookStatus string     `json:"webhook_status"`
	WebhookError  string     `json:"webhook_error,omitempty"`
	FiredAt       time.Time  `json:"fired_at"`
	AckedAt       *time.Time `json:"acked_at,omitempty"`
	AckedBy       string     `json:"acked_by,omitempty"`
}

type UpsertSmartAlertRuleRequest struct {
	CustomerID    string  `json:"customer_id"`
	CampaignID    string  `json:"campaign_id,omitempty"`
	Name          string  `json:"name"`
	Metric        string  `json:"metric"`
	Operator      string  `json:"operator"`
	Threshold     float64 `json:"threshold"`
	WindowMinutes int     `json:"window_minutes"`
	WebhookURL    string  `json:"webhook_url"`
	Enabled       bool    `json:"enabled"`
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Service == nil {
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
	mux.HandleFunc("GET /api/v1/smart-alerts/rules", limit(perm("campaigns:read", h.listRules)))
	mux.HandleFunc("POST /api/v1/smart-alerts/rules", limit(perm("campaigns:write", h.createRule)))
	mux.HandleFunc("PATCH /api/v1/smart-alerts/rules/{id}", limit(perm("campaigns:write", h.updateRule)))
	mux.HandleFunc("DELETE /api/v1/smart-alerts/rules/{id}", limit(perm("campaigns:write", h.deleteRule)))
	mux.HandleFunc("GET /api/v1/smart-alerts/history", limit(perm("campaigns:read", h.listHistory)))
	mux.HandleFunc("POST /api/v1/smart-alerts/events/{id}/ack", limit(perm("campaigns:write", h.ackEvent)))
}

func (h *HTTPHandlers) listRules(w http.ResponseWriter, r *http.Request) {
	custStr := r.URL.Query().Get("customer_id")
	custID, err := uuid.Parse(custStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	rules, err := h.Service.ListSmartAlertRules(r.Context(), custID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	if rules == nil {
		rules = []RuleDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, rules)
}

func (h *HTTPHandlers) createRule(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[UpsertRuleRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rule, err := h.Service.CreateSmartAlertRule(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, rule)
}

func (h *HTTPHandlers) updateRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[UpsertRuleRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rule, err := h.Service.UpdateSmartAlertRule(r.Context(), ruleID, req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, rule)
}

func (h *HTTPHandlers) deleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid rule id")
		return
	}
	if err := h.Service.DeleteSmartAlertRule(r.Context(), ruleID); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) listHistory(w http.ResponseWriter, r *http.Request) {
	custStr := r.URL.Query().Get("customer_id")
	custID, err := uuid.Parse(custStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	limit32, _ := coldpath.ParseAPIPaginationWith(r, 50, 200)
	limit := int(limit32)
	events, err := h.Service.ListSmartAlertHistory(r.Context(), custID, limit)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	if events == nil {
		events = []EventDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, events)
}

func (h *HTTPHandlers) ackEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid event id")
		return
	}
	actor := uuid.Nil
	if h.ResolveActorID != nil {
		actor = h.ResolveActorID(r.Context())
	}
	if err := h.Service.AckSmartAlertEvent(r.Context(), eventID, actor); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
