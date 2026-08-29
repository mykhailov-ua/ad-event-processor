package wizard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/importexport"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type campaignWizardService interface {
	CreateCampaignWizardSession(ctx context.Context, customerID uuid.UUID, templateKey string) (campaign.CampaignWizardSessionDTO, error)
	GetCampaignWizardSession(ctx context.Context, sessionID uuid.UUID) (campaign.CampaignWizardSessionDTO, error)
	UpdateCampaignWizardSessionStep(ctx context.Context, sessionID uuid.UUID, step string, payload json.RawMessage) (campaign.CampaignWizardSessionDTO, error)
	CommitCampaignWizardSession(ctx context.Context, sessionID uuid.UUID, idempotencyKey string, publish bool) (campaign.CampaignWizardCommitResult, error)
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

func listOnboardingTemplates(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	templates, err := campaign.ListOnboardingTemplates()
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

func getCampaignWizardSession(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
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

func postCampaignWizardSession(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
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
		handleCampaignWizardCreate(h, w, r, svc, req)
	case "update":
		handleCampaignWizardUpdate(h, w, r, svc, req)
	case "commit":
		handleCampaignWizardCommit(h, w, r, svc, req)
	default:
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "action must be create, update, or commit")
	}
}

func handleCampaignWizardCreate(h *campaign.CampaignsHTTPHandlers,
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

func handleCampaignWizardUpdate(h *campaign.CampaignsHTTPHandlers,
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

func handleCampaignWizardCommit(h *campaign.CampaignsHTTPHandlers,
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
	case errors.Is(err, campaign.ErrCampaignWizardSessionNotFound):
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, campaign.ErrCampaignWizardSessionExpired):
		httpresponse.Error(w, http.StatusGone, "SESSION_EXPIRED", err.Error())
	case errors.Is(err, campaign.ErrCampaignWizardIncomplete):
		httpresponse.Error(w, http.StatusUnprocessableEntity, "WIZARD_INCOMPLETE", err.Error())
	default:
		if errors.Is(err, campaign.ErrValidation) {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		if errors.Is(err, campaign.ErrForbidden) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
	}
}

type WizardHost interface {
	ApplyOnboardingTemplate(key string) (campaign.CampaignWizardStored, error)
	ImportBundledTemplate(ctx context.Context, schemaName string) error
	ImportCampaign(ctx context.Context, spec campaign.ImportCampaignSpec) (campaign.ImportCampaignResult, error)
	ApplyAffiliateNetworkTemplate(ctx context.Context, campaignID uuid.UUID, network, trackingDomain string) error
	PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (campaign.CampaignDTO, error)
	TrackingDomain(ctx context.Context, override string) string
	InboundTargetURL(ctx context.Context, schemaName, trackingDomain string) (string, error)
}

type WizardStore struct {
	pool *pgxpool.Pool
	host WizardHost
}

func NewWizardStore(pool *pgxpool.Pool, host WizardHost) *WizardStore {
	return &WizardStore{pool: pool, host: host}
}

func (st *WizardStore) poolOrNil() *pgxpool.Pool {
	if st == nil {
		return nil
	}
	return st.pool
}

const campaignWizardSessionTTL = 24 * time.Hour

const (
	WizardStepTrafficSource       = "traffic_source"
	WizardStepIntegrationTemplate = "integration_template"
	WizardStepFlowSkeleton        = "flow_skeleton"
	WizardStepBudget              = "budget"
	WizardStepReview              = "review"
)

const (
	wizardStepTrafficSource       = WizardStepTrafficSource
	wizardStepIntegrationTemplate = WizardStepIntegrationTemplate
	wizardStepFlowSkeleton        = WizardStepFlowSkeleton
	wizardStepBudget              = WizardStepBudget
	wizardStepReview              = WizardStepReview
)

var wizardStepOrder = []string{
	wizardStepTrafficSource,
	wizardStepIntegrationTemplate,
	wizardStepFlowSkeleton,
	wizardStepBudget,
	wizardStepReview,
}

type wizardSessionRow struct {
	ID             uuid.UUID
	CustomerID     uuid.UUID
	OwnerUserID    uuid.UUID
	CurrentStep    string
	CompletedSteps []string
	StepPayload    campaign.CampaignWizardStored
	ExpiresAt      time.Time
	UpdatedAt      time.Time
}

func (st *WizardStore) CreateCampaignWizardSession(ctx context.Context, customerID uuid.UUID, templateKey string) (campaign.CampaignWizardSessionDTO, error) {
	if st.poolOrNil() == nil {
		return campaign.CampaignWizardSessionDTO{}, fmt.Errorf("service unavailable")
	}
	if customerID == uuid.Nil {
		return campaign.CampaignWizardSessionDTO{}, campaign.ErrValidationf("customer_id is required")
	}
	if err := st.assertWizardCustomerAccess(ctx, customerID); err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}

	stored := campaign.CampaignWizardStored{}
	completed := []string{}
	currentStep := wizardStepTrafficSource
	templateKey = strings.TrimSpace(templateKey)
	if templateKey != "" {
		prefill, err := st.host.ApplyOnboardingTemplate(templateKey)
		if err != nil {
			return campaign.CampaignWizardSessionDTO{}, err
		}
		stored = prefill
		completed = append([]string(nil), wizardRequiredSteps()...)
		currentStep = wizardStepReview
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(campaignWizardSessionTTL)
	ownerID := wizardOwnerUserID(ctx)
	ownerParam := pgtype.UUID{}
	if ownerID != uuid.Nil {
		ownerParam = pgtype.UUID{Bytes: ownerID, Valid: true}
	}
	rawPayload, err := json.Marshal(stored)
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	rawCompleted, err := json.Marshal(completed)
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	_, err = st.poolOrNil().Exec(ctx, `
INSERT INTO campaign_wizard_sessions (
	id, customer_id, owner_user_id, current_step, completed_steps, step_payload, expires_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $8)`,
		sessionID, customerID, ownerParam, currentStep, rawCompleted, rawPayload, expiresAt, now,
	)
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	return st.GetCampaignWizardSession(ctx, sessionID)
}

func (st *WizardStore) GetCampaignWizardSession(ctx context.Context, sessionID uuid.UUID) (campaign.CampaignWizardSessionDTO, error) {
	row, err := st.loadCampaignWizardSession(ctx, sessionID)
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	return st.wizardSessionDTO(ctx, row)
}

func (st *WizardStore) UpdateCampaignWizardSessionStep(
	ctx context.Context,
	sessionID uuid.UUID,
	step string,
	payload json.RawMessage,
) (campaign.CampaignWizardSessionDTO, error) {
	if st.poolOrNil() == nil {
		return campaign.CampaignWizardSessionDTO{}, fmt.Errorf("service unavailable")
	}
	step = strings.TrimSpace(step)
	if step == "" {
		return campaign.CampaignWizardSessionDTO{}, campaign.ErrValidationf("step is required")
	}
	if !wizardStepKnown(step) {
		return campaign.CampaignWizardSessionDTO{}, campaign.ErrValidationf("unknown wizard step")
	}
	row, err := st.loadCampaignWizardSession(ctx, sessionID)
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	stored := row.StepPayload
	switch step {
	case wizardStepTrafficSource:
		var body campaign.CampaignWizardTrafficSourceStep
		if err := json.Unmarshal(payload, &body); err != nil {
			return campaign.CampaignWizardSessionDTO{}, campaign.ErrValidationf("invalid traffic_source payload")
		}
		if err := campaign.ValidateWizardTrafficSourceStep(body); err != nil {
			return campaign.CampaignWizardSessionDTO{}, err
		}
		stored.TrafficSource = body
	case wizardStepIntegrationTemplate:
		var body campaign.CampaignWizardIntegrationTemplateStep
		if err := json.Unmarshal(payload, &body); err != nil {
			return campaign.CampaignWizardSessionDTO{}, campaign.ErrValidationf("invalid integration_template payload")
		}
		if err := campaign.ValidateWizardIntegrationTemplateStep(body); err != nil {
			return campaign.CampaignWizardSessionDTO{}, err
		}
		stored.IntegrationTemplate = body
	case wizardStepFlowSkeleton:
		var body campaign.CampaignWizardFlowSkeletonStep
		if err := json.Unmarshal(payload, &body); err != nil {
			return campaign.CampaignWizardSessionDTO{}, campaign.ErrValidationf("invalid flow_skeleton payload")
		}
		if err := campaign.ValidateWizardFlowSkeletonStep(body); err != nil {
			return campaign.CampaignWizardSessionDTO{}, err
		}
		stored.FlowSkeleton = body
	case wizardStepBudget:
		var body campaign.CampaignWizardBudgetStep
		if err := json.Unmarshal(payload, &body); err != nil {
			return campaign.CampaignWizardSessionDTO{}, campaign.ErrValidationf("invalid budget payload")
		}
		if err := campaign.ValidateWizardBudgetStep(body); err != nil {
			return campaign.CampaignWizardSessionDTO{}, err
		}
		stored.Budget = body
	default:
		return campaign.CampaignWizardSessionDTO{}, campaign.ErrValidationf("step cannot be updated directly")
	}

	completed := appendWizardCompleted(row.CompletedSteps, step)
	current := wizardNextIncompleteStep(stored, completed)
	rawPayload, err := json.Marshal(stored)
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	rawCompleted, err := json.Marshal(completed)
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	now := time.Now().UTC()
	_, err = st.poolOrNil().Exec(ctx, `
UPDATE campaign_wizard_sessions
SET current_step = $2,
    completed_steps = $3::jsonb,
    step_payload = $4::jsonb,
    updated_at = $5
WHERE id = $1 AND expires_at > $5`,
		sessionID, current, rawCompleted, rawPayload, now,
	)
	if err != nil {
		return campaign.CampaignWizardSessionDTO{}, err
	}
	return st.GetCampaignWizardSession(ctx, sessionID)
}

func (st *WizardStore) CommitCampaignWizardSession(
	ctx context.Context,
	sessionID uuid.UUID,
	idempotencyKey string,
	publish bool,
) (campaign.CampaignWizardCommitResult, error) {
	if st.poolOrNil() == nil {
		return campaign.CampaignWizardCommitResult{}, fmt.Errorf("service unavailable")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return campaign.CampaignWizardCommitResult{}, campaign.ErrValidationf("idempotency_key is required")
	}
	row, err := st.loadCampaignWizardSession(ctx, sessionID)
	if err != nil {
		return campaign.CampaignWizardCommitResult{}, err
	}
	if !wizardReadyToCommit(row.CompletedSteps) {
		return campaign.CampaignWizardCommitResult{}, campaign.ErrCampaignWizardIncomplete
	}
	bundle, err := st.buildWizardImportBundle(ctx, row.StepPayload)
	if err != nil {
		return campaign.CampaignWizardCommitResult{}, err
	}
	imported, err := st.host.ImportCampaign(ctx, campaign.ImportCampaignSpec{
		CustomerID:     row.CustomerID,
		IdempotencyKey: idempotencyKey,
		Bundle:         bundle,
	})
	if err != nil {
		return campaign.CampaignWizardCommitResult{}, err
	}
	campaignID, err := uuid.Parse(imported.ID)
	if err != nil {
		return campaign.CampaignWizardCommitResult{}, err
	}
	if net := strings.TrimSpace(row.StepPayload.IntegrationTemplate.AffiliateNetwork); net != "" {
		err = st.host.ApplyAffiliateNetworkTemplate(ctx, campaignID, net, st.trackingDomain(ctx, row.StepPayload.IntegrationTemplate.TrackingDomain))
		if err != nil {
			return campaign.CampaignWizardCommitResult{}, err
		}
	}
	result := campaign.CampaignWizardCommitResult{Campaign: imported}
	if publish {
		_, pubErr := st.host.PublishCampaign(ctx, campaignID, false)
		if pubErr != nil {
			var blocked *campaign.CampaignPublishBlockedError
			if errors.As(pubErr, &blocked) {
				check := campaign.CampaignPublishCheckDTO{
					Valid:        false,
					FieldErrors:  blocked.FieldErrors,
					WarningSlugs: blocked.WarningSlugs,
				}
				result.PublishCheck = &check
				return result, nil
			}
			return result, pubErr
		}
		result.Published = true
	}
	_, _ = st.poolOrNil().Exec(ctx, `DELETE FROM campaign_wizard_sessions WHERE id = $1`, sessionID)
	return result, nil
}

func (st *WizardStore) loadCampaignWizardSession(ctx context.Context, sessionID uuid.UUID) (wizardSessionRow, error) {
	if sessionID == uuid.Nil {
		return wizardSessionRow{}, campaign.ErrValidationf("session_id is required")
	}
	var row wizardSessionRow
	var owner pgtype.UUID
	var completedRaw []byte
	var payloadRaw []byte
	err := st.poolOrNil().QueryRow(ctx, `
SELECT id, customer_id, owner_user_id, current_step, completed_steps, step_payload, expires_at, updated_at
FROM campaign_wizard_sessions
WHERE id = $1`, sessionID).Scan(
		&row.ID,
		&row.CustomerID,
		&owner,
		&row.CurrentStep,
		&completedRaw,
		&payloadRaw,
		&row.ExpiresAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return wizardSessionRow{}, campaign.ErrCampaignWizardSessionNotFound
		}
		return wizardSessionRow{}, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return wizardSessionRow{}, campaign.ErrCampaignWizardSessionExpired
	}
	if owner.Valid {
		row.OwnerUserID = uuid.UUID(owner.Bytes)
	}
	if len(completedRaw) > 0 {
		_ = json.Unmarshal(completedRaw, &row.CompletedSteps)
	}
	if len(payloadRaw) > 0 {
		_ = json.Unmarshal(payloadRaw, &row.StepPayload)
	}
	if err := st.assertWizardCustomerAccess(ctx, row.CustomerID); err != nil {
		return wizardSessionRow{}, err
	}
	if err := st.assertWizardSessionOwner(ctx, row); err != nil {
		return wizardSessionRow{}, err
	}
	return row, nil
}

func (st *WizardStore) wizardSessionDTO(ctx context.Context, row wizardSessionRow) (campaign.CampaignWizardSessionDTO, error) {
	steps := campaign.CampaignWizardStepsDTO{
		TrafficSource:       wizardTrafficSourcePtr(row.StepPayload.TrafficSource),
		IntegrationTemplate: wizardIntegrationTemplatePtr(row.StepPayload.IntegrationTemplate),
		FlowSkeleton:        wizardFlowSkeletonPtr(row.StepPayload.FlowSkeleton),
		Budget:              wizardBudgetPtr(row.StepPayload.Budget),
	}
	dto := campaign.CampaignWizardSessionDTO{
		SessionID:      row.ID.String(),
		CustomerID:     row.CustomerID.String(),
		CurrentStep:    row.CurrentStep,
		CompletedSteps: append([]string(nil), row.CompletedSteps...),
		Steps:          steps,
		ReadyToCommit:  wizardReadyToCommit(row.CompletedSteps),
		ExpiresAt:      row.ExpiresAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if dto.ReadyToCommit || row.CurrentStep == wizardStepReview {
		review, err := st.buildWizardReview(ctx, row.StepPayload)
		if err != nil {
			return campaign.CampaignWizardSessionDTO{}, err
		}
		dto.Review = &review
	}
	return dto, nil
}

func (st *WizardStore) buildWizardReview(ctx context.Context, stored campaign.CampaignWizardStored) (campaign.CampaignWizardReviewDTO, error) {
	var warnings []string
	if len(stored.Budget.TargetCountries) == 0 {
		warnings = append(warnings, "target_countries_empty")
	}
	targetURL, err := st.inboundTargetURL(ctx, stored.IntegrationTemplate.IntegrationSchema, stored.IntegrationTemplate.TrackingDomain)
	if err != nil {
		return campaign.CampaignWizardReviewDTO{}, err
	}
	if strings.TrimSpace(targetURL) == "" {
		warnings = append(warnings, "target_url_empty")
	}
	preview := campaign.CampaignWizardPreviewDTO{
		CampaignName:      strings.TrimSpace(stored.TrafficSource.Name),
		TrafficTemplateID: strings.TrimSpace(stored.TrafficSource.TrafficTemplateID),
		IntegrationSchema: strings.TrimSpace(stored.IntegrationTemplate.IntegrationSchema),
		FlowName:          strings.TrimSpace(stored.FlowSkeleton.FlowName),
		BudgetLimitMicro:  stored.Budget.BudgetLimitMicro,
		TargetURL:         targetURL,
	}
	return campaign.CampaignWizardReviewDTO{Preview: preview, WarningSlugs: warnings}, nil
}

func (st *WizardStore) buildWizardImportBundle(ctx context.Context, stored campaign.CampaignWizardStored) (campaign.CampaignExportBundle, error) {
	if !wizardPayloadComplete(stored) {
		return campaign.CampaignExportBundle{}, campaign.ErrCampaignWizardIncomplete
	}
	schemaName := strings.TrimSpace(stored.IntegrationTemplate.IntegrationSchema)
	if entry, ok := integrationschema.FindCatalogEntry(schemaName); ok {
		if err := st.host.ImportBundledTemplate(ctx, entry.Name); err != nil {
			return campaign.CampaignExportBundle{}, err
		}
	}
	targetURL, err := st.inboundTargetURL(ctx, schemaName, stored.IntegrationTemplate.TrackingDomain)
	if err != nil {
		return campaign.CampaignExportBundle{}, err
	}
	flowName := strings.TrimSpace(stored.FlowSkeleton.FlowName)
	landerRef := "lander-1"
	offerRef := "offer-1"
	bundle := campaign.CampaignExportBundle{
		ExportVersion:         campaign.CampaignExportVersion,
		IntegrationSchemaName: schemaName,
		Campaign: campaign.CampaignExportCampaign{
			Name:              strings.TrimSpace(stored.TrafficSource.Name),
			BudgetLimitMicro:  stored.Budget.BudgetLimitMicro,
			Timezone:          importexport.DefaultTimezone(stored.Budget.Timezone),
			TargetCountries:   append([]string(nil), stored.Budget.TargetCountries...),
			TargetURL:         targetURL,
			TrafficTemplateID: strings.TrimSpace(stored.TrafficSource.TrafficTemplateID),
			ClickQueryParams:  stored.TrafficSource.ClickQueryParams,
		},
		Flow: &campaign.CampaignExportFlow{
			Name: flowName,
			Paths: []campaign.CampaignExportFlowPath{{
				Weight: 100,
				Landers: []campaign.CampaignExportFlowLanderRef{{
					Ref:    landerRef,
					Weight: 100,
				}},
				Offers: []campaign.CampaignExportFlowOfferRef{{
					Ref:    offerRef,
					Weight: 100,
				}},
			}},
		},
		Landers: []campaign.CampaignExportLander{{
			Ref:  landerRef,
			Name: strings.TrimSpace(stored.FlowSkeleton.Lander.Name),
			URL:  strings.TrimSpace(stored.FlowSkeleton.Lander.URL),
		}},
		Offers: []campaign.CampaignExportOffer{{
			Ref:  offerRef,
			Name: strings.TrimSpace(stored.FlowSkeleton.Offer.Name),
			URL:  strings.TrimSpace(stored.FlowSkeleton.Offer.URL),
		}},
	}
	return bundle, nil
}

func wizardStepKnown(step string) bool {
	for _, candidate := range wizardStepOrder {
		if candidate == step {
			return true
		}
	}
	return false
}

func wizardRequiredSteps() []string {
	return []string{
		wizardStepTrafficSource,
		wizardStepIntegrationTemplate,
		wizardStepFlowSkeleton,
		wizardStepBudget,
	}
}

func wizardReadyToCommit(completed []string) bool {
	required := wizardRequiredSteps()
	seen := make(map[string]struct{}, len(completed))
	for _, step := range completed {
		seen[step] = struct{}{}
	}
	for _, step := range required {
		if _, ok := seen[step]; !ok {
			return false
		}
	}
	return true
}

func wizardPayloadComplete(stored campaign.CampaignWizardStored) bool {
	return strings.TrimSpace(stored.TrafficSource.Name) != "" &&
		strings.TrimSpace(stored.IntegrationTemplate.IntegrationSchema) != "" &&
		strings.TrimSpace(stored.FlowSkeleton.FlowName) != "" &&
		strings.TrimSpace(stored.FlowSkeleton.Lander.URL) != "" &&
		strings.TrimSpace(stored.FlowSkeleton.Offer.URL) != "" &&
		stored.Budget.BudgetLimitMicro > 0
}

func appendWizardCompleted(existing []string, step string) []string {
	for _, item := range existing {
		if item == step {
			return existing
		}
	}
	return append(existing, step)
}

func wizardNextIncompleteStep(stored campaign.CampaignWizardStored, completed []string) string {
	if wizardReadyToCommit(completed) {
		return wizardStepReview
	}
	for _, step := range wizardRequiredSteps() {
		if !wizardStepCompleted(step, stored, completed) {
			return step
		}
	}
	return wizardStepReview
}

func wizardStepCompleted(step string, stored campaign.CampaignWizardStored, completed []string) bool {
	for _, item := range completed {
		if item == step {
			return true
		}
	}
	switch step {
	case wizardStepTrafficSource:
		return campaign.ValidateWizardTrafficSourceStep(stored.TrafficSource) == nil
	case wizardStepIntegrationTemplate:
		return campaign.ValidateWizardIntegrationTemplateStep(stored.IntegrationTemplate) == nil
	case wizardStepFlowSkeleton:
		return campaign.ValidateWizardFlowSkeletonStep(stored.FlowSkeleton) == nil
	case wizardStepBudget:
		return campaign.ValidateWizardBudgetStep(stored.Budget) == nil
	default:
		return false
	}
}

func wizardOwnerUserID(ctx context.Context) uuid.UUID {
	u, ok := authz.GetUser(ctx)
	if !ok {
		return uuid.Nil
	}
	if authz.NormalizeRole(u.Role) == authz.RoleMediaBuyer {
		return u.UserID
	}
	return uuid.Nil
}

func (st *WizardStore) assertWizardCustomerAccess(ctx context.Context, customerID uuid.UUID) error {
	if customerID == uuid.Nil {
		return campaign.ErrValidationf("customer_id is required")
	}
	u, ok := authz.GetUser(ctx)
	if !ok || authz.NormalizeRole(u.Role) != authz.RoleMediaBuyer {
		return nil
	}
	if u.CustomerID != uuid.Nil && u.CustomerID != customerID {
		return campaign.ErrForbidden
	}
	return nil
}

func (st *WizardStore) assertWizardSessionOwner(ctx context.Context, row wizardSessionRow) error {
	u, ok := authz.GetUser(ctx)
	if !ok || authz.NormalizeRole(u.Role) != authz.RoleMediaBuyer {
		return nil
	}
	if row.OwnerUserID == uuid.Nil {
		return nil
	}
	if row.OwnerUserID != u.UserID {
		return campaign.ErrForbidden
	}
	return nil
}

func wizardTrafficSourcePtr(step campaign.CampaignWizardTrafficSourceStep) *campaign.CampaignWizardTrafficSourceStep {
	if strings.TrimSpace(step.Name) == "" {
		return nil
	}
	copyStep := step
	return &copyStep
}

func wizardIntegrationTemplatePtr(step campaign.CampaignWizardIntegrationTemplateStep) *campaign.CampaignWizardIntegrationTemplateStep {
	if strings.TrimSpace(step.IntegrationSchema) == "" {
		return nil
	}
	copyStep := step
	return &copyStep
}

func wizardFlowSkeletonPtr(step campaign.CampaignWizardFlowSkeletonStep) *campaign.CampaignWizardFlowSkeletonStep {
	if strings.TrimSpace(step.FlowName) == "" {
		return nil
	}
	copyStep := step
	return &copyStep
}

func wizardBudgetPtr(step campaign.CampaignWizardBudgetStep) *campaign.CampaignWizardBudgetStep {
	if step.BudgetLimitMicro <= 0 {
		return nil
	}
	copyStep := step
	return &copyStep
}

func (st *WizardStore) inboundTargetURL(ctx context.Context, schemaName, trackingDomain string) (string, error) {
	if st == nil || st.host == nil {
		return "", fmt.Errorf("service unavailable")
	}
	return st.host.InboundTargetURL(ctx, schemaName, trackingDomain)
}

func (st *WizardStore) trackingDomain(ctx context.Context, override string) string {
	if st == nil || st.host == nil {
		return ""
	}
	return st.host.TrackingDomain(ctx, override)
}
