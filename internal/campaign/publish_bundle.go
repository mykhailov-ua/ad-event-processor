package campaign

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"ad-event-processor/internal/controlplane/authz"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/proxyupstream"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PublishGateEvalInput struct {
	CampaignID          uuid.UUID
	Name                string
	BudgetLimit         int64
	CurrentSpend        int64
	TargetCountries     []string
	TargetURL           string
	ClickQueryParams    map[string]string
	ClickDelivery       string
	ProxyUpstreamURL    string
	AllowHTTPInsecure   bool
	FlowMissing         bool
	FlowPathError       string
	IntegrationHealth   IntegrationHealthDTO
	IntegrationHealthOK bool
}

func EvaluatePublishBlocked(ctx context.Context, input PublishGateEvalInput) *CampaignPublishBlockedError {
	fieldErrors := make(map[string]string)
	var warningSlugs []string

	if strings.TrimSpace(input.Name) == "" {
		fieldErrors["name"] = "name is required"
	}
	if input.BudgetLimit <= 0 {
		fieldErrors["budget_limit"] = "budget must be positive"
	}
	if input.BudgetLimit < input.CurrentSpend {
		fieldErrors["budget_limit"] = "budget_limit cannot be below current spend"
	}
	if len(input.TargetCountries) == 0 {
		warningSlugs = append(warningSlugs, "target_countries_empty")
	}

	if input.FlowMissing {
		fieldErrors["flow_id"] = "campaign flow is required"
	} else if input.FlowPathError != "" {
		fieldErrors["flow_paths"] = input.FlowPathError
	}

	targetURL := strings.TrimSpace(input.TargetURL)
	if targetURL == "" {
		warningSlugs = append(warningSlugs, "target_url_empty")
	} else {
		macroCtx := PreviewContext(input.CampaignID.String(), PreviewRequest{
			Sub1:    "preview",
			Country: "US",
		})
		_, unresolved := Expand(targetURL, macroCtx)
		if len(unresolved) > 0 {
			fieldErrors["target_url"] = fmt.Sprintf("unresolved macros: %s", strings.Join(unresolved, ", "))
		}
		if params := input.ClickQueryParams; len(params) > 0 {
			for key, value := range params {
				_, paramUnresolved := Expand(value, macroCtx)
				if len(paramUnresolved) > 0 {
					fieldErrors["click_query_params."+key] = fmt.Sprintf("unresolved macros: %s", strings.Join(paramUnresolved, ", "))
				}
			}
		}
		if strings.HasPrefix(strings.ToLower(targetURL), "http://") {
			warningSlugs = append(warningSlugs, "macro_click_url_uses_http")
		}
	}

	if err := proxyupstream.ValidateDeliveryPair(ctx, input.ClickDelivery, input.ProxyUpstreamURL, input.AllowHTTPInsecure); err != nil {
		fieldErrors["click_delivery"] = err.Error()
	}

	if input.IntegrationHealthOK {
		for _, healthRow := range input.IntegrationHealth.Rows {
			if healthRow.Status == string(IntegrationHealthFail) {
				key := "integration_" + strings.TrimSpace(healthRow.Slug)
				if key == "integration_" {
					key = "integration"
				}
				fieldErrors[key] = healthRow.Message
			}
		}
	}

	if len(fieldErrors) == 0 {
		return nil
	}
	return &CampaignPublishBlockedError{
		FieldErrors:  fieldErrors,
		WarningSlugs: warningSlugs,
	}
}

func EnforcePublishGate(ctx context.Context, fx Effects, pool *pgxpool.Pool, campaignID uuid.UUID, row db.Campaign, force bool) error {
	if force {
		return AuditPublishForce(ctx, fx, pool, campaignID)
	}
	blocked, err := CollectPublishBlocked(ctx, fx, campaignID, row)
	if err != nil {
		return err
	}
	if blocked != nil {
		return blocked
	}
	return nil
}

func AuditPublishForce(ctx context.Context, fx Effects, pool *pgxpool.Pool, campaignID uuid.UUID) error {
	if fx == nil || pool == nil {
		return errServiceUnavailable()
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		var uid uuid.UUID
		if user, ok := authz.GetUser(ctx); ok {
			uid = user.UserID
		}
		fx.AuditLog(ctx, q, uid, "PUBLISH_FORCE", "campaign", &campaignID, auditReasonChange{Reason: "publish_gate_bypass"}, nil)
		return nil
	})
}

func (h *CampaignsHTTPHandlers) registerCampaignPublishRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	write := []string{"campaigns:write"}
	mux.HandleFunc("POST /api/v1/campaigns/{id}/publish", limit(perm(write, h.postCampaignPublish)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/publish-check", limit(perm(write, h.getCampaignPublishCheck)))
}

type campaignPublisher interface {
	PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (CampaignDTO, error)
	EvaluateCampaignPublish(ctx context.Context, campaignID uuid.UUID) (CampaignPublishCheckDTO, error)
}

func (h *CampaignsHTTPHandlers) postCampaignPublish(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	publisher, ok := h.Campaigns.(campaignPublisher)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "publish not configured")
		return
	}
	force := ParsePublishForceQuery(r.URL.Query().Get("force"))
	if force && !CanForceCampaignPublish(r.Context()) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "force publish requires admin role")
		return
	}
	updated, err := publisher.PublishCampaign(r.Context(), campaignID, force)
	if err != nil {
		WriteCampaignPublishError(w, err, h.WriteServiceError)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h *CampaignsHTTPHandlers) getCampaignPublishCheck(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	publisher, ok := h.Campaigns.(campaignPublisher)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "publish check not configured")
		return
	}
	result, err := publisher.EvaluateCampaignPublish(r.Context(), campaignID)
	if err != nil {
		WriteCampaignPublishError(w, err, h.WriteServiceError)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

type CampaignPublishBlockedError struct {
	FieldErrors  map[string]string `json:"field_errors"`
	WarningSlugs []string          `json:"warning_slugs,omitempty"`
}

func (e *CampaignPublishBlockedError) Error() string {
	return ErrCampaignPublishBlocked.Error()
}

func (e *CampaignPublishBlockedError) Is(target error) bool {
	return errors.Is(e, ErrCampaignPublishBlocked)
}

func ParsePublishForceQuery(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func CanForceCampaignPublish(ctx context.Context) bool {
	user, ok := authz.GetUser(ctx)
	return ok && authz.NormalizeRole(user.Role) == authz.RoleAdmin
}

func WriteCampaignPublishError(w http.ResponseWriter, err error, writeServiceError func(http.ResponseWriter, error)) {
	var blocked *CampaignPublishBlockedError
	if errors.As(err, &blocked) {
		httpresponse.JSON(w, http.StatusUnprocessableEntity, blocked)
		return
	}
	if writeServiceError != nil {
		writeServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
}
