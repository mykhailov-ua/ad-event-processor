package controlplane

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const fraudDecisionDisclaimer = "Decision as of last scorer window; replay uses stored features and shadow ML score."

// FraudDecisionDisclaimer returns the operator-facing replay disclaimer.
func FraudDecisionDisclaimer() string {
	return fraudDecisionDisclaimer
}

type FraudDecisionDTO struct {
	IPHash              string                 `json:"ip_hash"`
	CampaignID          string                 `json:"campaign_id"`
	WindowStart         string                 `json:"window_start"`
	EvaluatedAt         string                 `json:"evaluated_at"`
	Disclaimer          string                 `json:"disclaimer"`
	Tier                string                 `json:"tier"`
	Score               int                    `json:"score"`
	MLProbability       float64                `json:"ml_probability"`
	AdjustedProbability float64                `json:"adjusted_probability"`
	ResidentialProxy    bool                   `json:"residential_proxy"`
	StructuralFraud     bool                   `json:"structural_fraud"`
	FPGuardApplied      bool                   `json:"fp_guard_applied"`
	ModelScore          *float64               `json:"model_score,omitempty"`
	ModelName           string                 `json:"model_name,omitempty"`
	ScoreMissing        bool                   `json:"score_missing"`
	Features            map[string]float64     `json:"features"`
	CampaignThresholds  FraudTierThresholdsDTO `json:"campaign_thresholds"`
}

type FraudDecisionsService interface {
	ExplainFraudDecision(ctx context.Context, customerID uuid.UUID, ipHash string, campaignID *uuid.UUID, hours int) (FraudDecisionDTO, error)
}

func (fraud *FraudHTTPHandlers) registerFraudDecisionRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	if fraud == nil || fraud.Decisions == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/decisions", limit(perm("audit:read", fraud.getFraudDecision)))
}

func (fraud *FraudHTTPHandlers) getFraudDecision(w http.ResponseWriter, r *http.Request) {
	customerID, ok := fraud.resolveCustomerID(w, r)
	if !ok {
		return
	}
	if fraud.AllowFraudDecision != nil && !fraud.AllowFraudDecision(customerID.String()) {
		httpresponse.Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "fraud decision lookup rate limit exceeded")
		return
	}

	ipHash := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("ip_hash")))
	if !validMLIPHashHex(ipHash) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash must be 32 hex characters")
		return
	}

	hours := 24
	if raw := strings.TrimSpace(r.URL.Query().Get("hours")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid hours")
			return
		}
		hours = parsed
	}
	if hours > fraudExplainMaxHours {
		hours = fraudExplainMaxHours
	}

	var campaignID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("campaign_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		if fraud.AuthorizeCampaignAccess != nil {
			if err := fraud.AuthorizeCampaignAccess(r, id); err != nil {
				fraud.writeServiceError(w, err)
				return
			}
		}
		campaignID = &id
	}

	out, err := fraud.Decisions.ExplainFraudDecision(r.Context(), customerID, ipHash, campaignID, hours)
	if err != nil {
		fraud.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

const fraudExplainMaxHours = 168
