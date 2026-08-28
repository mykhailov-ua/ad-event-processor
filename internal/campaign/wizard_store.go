package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/integrationschema"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	StepPayload    CampaignWizardStored
	ExpiresAt      time.Time
	UpdatedAt      time.Time
}

func (st *WizardStore) CreateCampaignWizardSession(ctx context.Context, customerID uuid.UUID, templateKey string) (CampaignWizardSessionDTO, error) {
	if st.poolOrNil() == nil {
		return CampaignWizardSessionDTO{}, fmt.Errorf("service unavailable")
	}
	if customerID == uuid.Nil {
		return CampaignWizardSessionDTO{}, errValidation("customer_id is required")
	}
	if err := st.assertWizardCustomerAccess(ctx, customerID); err != nil {
		return CampaignWizardSessionDTO{}, err
	}

	stored := CampaignWizardStored{}
	completed := []string{}
	currentStep := wizardStepTrafficSource
	templateKey = strings.TrimSpace(templateKey)
	if templateKey != "" {
		prefill, err := st.host.ApplyOnboardingTemplate(templateKey)
		if err != nil {
			return CampaignWizardSessionDTO{}, err
		}
		stored = prefill
		completed = append([]string(nil), wizardRequiredSteps()...)
		currentStep = wizardStepReview
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return CampaignWizardSessionDTO{}, err
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
		return CampaignWizardSessionDTO{}, err
	}
	rawCompleted, err := json.Marshal(completed)
	if err != nil {
		return CampaignWizardSessionDTO{}, err
	}
	_, err = st.poolOrNil().Exec(ctx, `
INSERT INTO campaign_wizard_sessions (
	id, customer_id, owner_user_id, current_step, completed_steps, step_payload, expires_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $8)`,
		sessionID, customerID, ownerParam, currentStep, rawCompleted, rawPayload, expiresAt, now,
	)
	if err != nil {
		return CampaignWizardSessionDTO{}, err
	}
	return st.GetCampaignWizardSession(ctx, sessionID)
}

func (st *WizardStore) GetCampaignWizardSession(ctx context.Context, sessionID uuid.UUID) (CampaignWizardSessionDTO, error) {
	row, err := st.loadCampaignWizardSession(ctx, sessionID)
	if err != nil {
		return CampaignWizardSessionDTO{}, err
	}
	return st.wizardSessionDTO(ctx, row)
}

func (st *WizardStore) UpdateCampaignWizardSessionStep(
	ctx context.Context,
	sessionID uuid.UUID,
	step string,
	payload json.RawMessage,
) (CampaignWizardSessionDTO, error) {
	if st.poolOrNil() == nil {
		return CampaignWizardSessionDTO{}, fmt.Errorf("service unavailable")
	}
	step = strings.TrimSpace(step)
	if step == "" {
		return CampaignWizardSessionDTO{}, errValidation("step is required")
	}
	if !wizardStepKnown(step) {
		return CampaignWizardSessionDTO{}, errValidation("unknown wizard step")
	}
	row, err := st.loadCampaignWizardSession(ctx, sessionID)
	if err != nil {
		return CampaignWizardSessionDTO{}, err
	}
	stored := row.StepPayload
	switch step {
	case wizardStepTrafficSource:
		var body CampaignWizardTrafficSourceStep
		if err := json.Unmarshal(payload, &body); err != nil {
			return CampaignWizardSessionDTO{}, errValidation("invalid traffic_source payload")
		}
		if err := validateWizardTrafficSourceStep(body); err != nil {
			return CampaignWizardSessionDTO{}, err
		}
		stored.TrafficSource = body
	case wizardStepIntegrationTemplate:
		var body CampaignWizardIntegrationTemplateStep
		if err := json.Unmarshal(payload, &body); err != nil {
			return CampaignWizardSessionDTO{}, errValidation("invalid integration_template payload")
		}
		if err := validateWizardIntegrationTemplateStep(body); err != nil {
			return CampaignWizardSessionDTO{}, err
		}
		stored.IntegrationTemplate = body
	case wizardStepFlowSkeleton:
		var body CampaignWizardFlowSkeletonStep
		if err := json.Unmarshal(payload, &body); err != nil {
			return CampaignWizardSessionDTO{}, errValidation("invalid flow_skeleton payload")
		}
		if err := validateWizardFlowSkeletonStep(body); err != nil {
			return CampaignWizardSessionDTO{}, err
		}
		stored.FlowSkeleton = body
	case wizardStepBudget:
		var body CampaignWizardBudgetStep
		if err := json.Unmarshal(payload, &body); err != nil {
			return CampaignWizardSessionDTO{}, errValidation("invalid budget payload")
		}
		if err := validateWizardBudgetStep(body); err != nil {
			return CampaignWizardSessionDTO{}, err
		}
		stored.Budget = body
	default:
		return CampaignWizardSessionDTO{}, errValidation("step cannot be updated directly")
	}

	completed := appendWizardCompleted(row.CompletedSteps, step)
	current := wizardNextIncompleteStep(stored, completed)
	rawPayload, err := json.Marshal(stored)
	if err != nil {
		return CampaignWizardSessionDTO{}, err
	}
	rawCompleted, err := json.Marshal(completed)
	if err != nil {
		return CampaignWizardSessionDTO{}, err
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
		return CampaignWizardSessionDTO{}, err
	}
	return st.GetCampaignWizardSession(ctx, sessionID)
}

func (st *WizardStore) CommitCampaignWizardSession(
	ctx context.Context,
	sessionID uuid.UUID,
	idempotencyKey string,
	publish bool,
) (CampaignWizardCommitResult, error) {
	if st.poolOrNil() == nil {
		return CampaignWizardCommitResult{}, fmt.Errorf("service unavailable")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return CampaignWizardCommitResult{}, errValidation("idempotency_key is required")
	}
	row, err := st.loadCampaignWizardSession(ctx, sessionID)
	if err != nil {
		return CampaignWizardCommitResult{}, err
	}
	if !wizardReadyToCommit(row.CompletedSteps) {
		return CampaignWizardCommitResult{}, ErrCampaignWizardIncomplete
	}
	bundle, err := st.buildWizardImportBundle(ctx, row.StepPayload)
	if err != nil {
		return CampaignWizardCommitResult{}, err
	}
	imported, err := st.host.ImportCampaign(ctx, ImportCampaignSpec{
		CustomerID:     row.CustomerID,
		IdempotencyKey: idempotencyKey,
		Bundle:         bundle,
	})
	if err != nil {
		return CampaignWizardCommitResult{}, err
	}
	campaignID, err := uuid.Parse(imported.ID)
	if err != nil {
		return CampaignWizardCommitResult{}, err
	}
	if net := strings.TrimSpace(row.StepPayload.IntegrationTemplate.AffiliateNetwork); net != "" {
		err = st.host.ApplyAffiliateNetworkTemplate(ctx, campaignID, net, st.trackingDomain(ctx, row.StepPayload.IntegrationTemplate.TrackingDomain))
		if err != nil {
			return CampaignWizardCommitResult{}, err
		}
	}
	result := CampaignWizardCommitResult{Campaign: imported}
	if publish {
		_, pubErr := st.host.PublishCampaign(ctx, campaignID, false)
		if pubErr != nil {
			var blocked *CampaignPublishBlockedError
			if errors.As(pubErr, &blocked) {
				check := CampaignPublishCheckDTO{
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
		return wizardSessionRow{}, errValidation("session_id is required")
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
			return wizardSessionRow{}, ErrCampaignWizardSessionNotFound
		}
		return wizardSessionRow{}, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return wizardSessionRow{}, ErrCampaignWizardSessionExpired
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

func (st *WizardStore) wizardSessionDTO(ctx context.Context, row wizardSessionRow) (CampaignWizardSessionDTO, error) {
	steps := CampaignWizardStepsDTO{
		TrafficSource:       wizardTrafficSourcePtr(row.StepPayload.TrafficSource),
		IntegrationTemplate: wizardIntegrationTemplatePtr(row.StepPayload.IntegrationTemplate),
		FlowSkeleton:        wizardFlowSkeletonPtr(row.StepPayload.FlowSkeleton),
		Budget:              wizardBudgetPtr(row.StepPayload.Budget),
	}
	dto := CampaignWizardSessionDTO{
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
			return CampaignWizardSessionDTO{}, err
		}
		dto.Review = &review
	}
	return dto, nil
}

func (st *WizardStore) buildWizardReview(ctx context.Context, stored CampaignWizardStored) (CampaignWizardReviewDTO, error) {
	var warnings []string
	if len(stored.Budget.TargetCountries) == 0 {
		warnings = append(warnings, "target_countries_empty")
	}
	targetURL, err := st.inboundTargetURL(ctx, stored.IntegrationTemplate.IntegrationSchema, stored.IntegrationTemplate.TrackingDomain)
	if err != nil {
		return CampaignWizardReviewDTO{}, err
	}
	if strings.TrimSpace(targetURL) == "" {
		warnings = append(warnings, "target_url_empty")
	}
	preview := CampaignWizardPreviewDTO{
		CampaignName:      strings.TrimSpace(stored.TrafficSource.Name),
		TrafficTemplateID: strings.TrimSpace(stored.TrafficSource.TrafficTemplateID),
		IntegrationSchema: strings.TrimSpace(stored.IntegrationTemplate.IntegrationSchema),
		FlowName:          strings.TrimSpace(stored.FlowSkeleton.FlowName),
		BudgetLimitMicro:  stored.Budget.BudgetLimitMicro,
		TargetURL:         targetURL,
	}
	return CampaignWizardReviewDTO{Preview: preview, WarningSlugs: warnings}, nil
}

func (st *WizardStore) buildWizardImportBundle(ctx context.Context, stored CampaignWizardStored) (CampaignExportBundle, error) {
	if !wizardPayloadComplete(stored) {
		return CampaignExportBundle{}, ErrCampaignWizardIncomplete
	}
	schemaName := strings.TrimSpace(stored.IntegrationTemplate.IntegrationSchema)
	if entry, ok := integrationschema.FindCatalogEntry(schemaName); ok {
		if err := st.host.ImportBundledTemplate(ctx, entry.Name); err != nil {
			return CampaignExportBundle{}, err
		}
	}
	targetURL, err := st.inboundTargetURL(ctx, schemaName, stored.IntegrationTemplate.TrackingDomain)
	if err != nil {
		return CampaignExportBundle{}, err
	}
	flowName := strings.TrimSpace(stored.FlowSkeleton.FlowName)
	landerRef := "lander-1"
	offerRef := "offer-1"
	bundle := CampaignExportBundle{
		ExportVersion:         CampaignExportVersion,
		IntegrationSchemaName: schemaName,
		Campaign: CampaignExportCampaign{
			Name:              strings.TrimSpace(stored.TrafficSource.Name),
			BudgetLimitMicro:  stored.Budget.BudgetLimitMicro,
			Timezone:          DefaultTimezone(stored.Budget.Timezone),
			TargetCountries:   append([]string(nil), stored.Budget.TargetCountries...),
			TargetURL:         targetURL,
			TrafficTemplateID: strings.TrimSpace(stored.TrafficSource.TrafficTemplateID),
			ClickQueryParams:  stored.TrafficSource.ClickQueryParams,
		},
		Flow: &CampaignExportFlow{
			Name: flowName,
			Paths: []CampaignExportFlowPath{{
				Weight: 100,
				Landers: []CampaignExportFlowLanderRef{{
					Ref:    landerRef,
					Weight: 100,
				}},
				Offers: []CampaignExportFlowOfferRef{{
					Ref:    offerRef,
					Weight: 100,
				}},
			}},
		},
		Landers: []CampaignExportLander{{
			Ref:  landerRef,
			Name: strings.TrimSpace(stored.FlowSkeleton.Lander.Name),
			URL:  strings.TrimSpace(stored.FlowSkeleton.Lander.URL),
		}},
		Offers: []CampaignExportOffer{{
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

func wizardPayloadComplete(stored CampaignWizardStored) bool {
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

func wizardNextIncompleteStep(stored CampaignWizardStored, completed []string) string {
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

func wizardStepCompleted(step string, stored CampaignWizardStored, completed []string) bool {
	for _, item := range completed {
		if item == step {
			return true
		}
	}
	switch step {
	case wizardStepTrafficSource:
		return validateWizardTrafficSourceStep(stored.TrafficSource) == nil
	case wizardStepIntegrationTemplate:
		return validateWizardIntegrationTemplateStep(stored.IntegrationTemplate) == nil
	case wizardStepFlowSkeleton:
		return validateWizardFlowSkeletonStep(stored.FlowSkeleton) == nil
	case wizardStepBudget:
		return validateWizardBudgetStep(stored.Budget) == nil
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
		return errValidation("customer_id is required")
	}
	u, ok := authz.GetUser(ctx)
	if !ok || authz.NormalizeRole(u.Role) != authz.RoleMediaBuyer {
		return nil
	}
	if u.CustomerID != uuid.Nil && u.CustomerID != customerID {
		return ErrForbidden
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
		return ErrForbidden
	}
	return nil
}

func wizardTrafficSourcePtr(step CampaignWizardTrafficSourceStep) *CampaignWizardTrafficSourceStep {
	if strings.TrimSpace(step.Name) == "" {
		return nil
	}
	copyStep := step
	return &copyStep
}

func wizardIntegrationTemplatePtr(step CampaignWizardIntegrationTemplateStep) *CampaignWizardIntegrationTemplateStep {
	if strings.TrimSpace(step.IntegrationSchema) == "" {
		return nil
	}
	copyStep := step
	return &copyStep
}

func wizardFlowSkeletonPtr(step CampaignWizardFlowSkeletonStep) *CampaignWizardFlowSkeletonStep {
	if strings.TrimSpace(step.FlowName) == "" {
		return nil
	}
	copyStep := step
	return &copyStep
}

func wizardBudgetPtr(step CampaignWizardBudgetStep) *CampaignWizardBudgetStep {
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
