package controlplane

import (
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/internal/supply"
	"ad-event-processor/pkg/httpresponse"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var errLicenseWatcherUnavailable = errors.New("license watcher not configured")

const defaultLicenseRevokePoll = 30 * time.Second

var activeLicenseWatcher atomic.Pointer[licensing.LicenseWatcher]

func setLicenseWatcher(w *licensing.LicenseWatcher) {
	if w != nil {
		activeLicenseWatcher.Store(w)
	}
}

func startLicenseWatcher(ctx context.Context, pool *pgxpool.Pool, redisShards []redis.UniversalClient, pubKey ed25519.PublicKey, svc *Service) error {
	watcher := licensing.NewLicenseWatcher(pool, shardadmin.PickHealthyControlShard(redisShards), pubKey)
	watcher.SetControlRedisShards(redisShards)
	setLicenseWatcher(watcher)
	if err := watcher.Start(ctx); err != nil {
		return err
	}
	if config.LicenseRequiredFromEnv() {
		state, _ := watcher.GetState()
		if state == licensing.StateExpired || state == licensing.StateRevoked {
			slog.Warn("license required but ingest not allowed at startup", "state", state)
		}
	}
	if svc != nil {
		svc.StartBackgroundWorker(func() {
			<-ctx.Done()
		})
	}
	slog.Info("started license watcher", "mode", config.LicenseEnv("MODE"))
	return nil
}

func licenseIngestReady() bool {
	if !config.LicenseRequiredFromEnv() {
		return true
	}
	w := activeLicenseWatcher.Load()
	if w == nil {
		if licensing.SeedCouplingRequired() {
			return licensing.FeatureSeedValid()
		}
		return true
	}
	state, _ := w.GetState()
	if state == licensing.StateExpired || state == licensing.StateRevoked {
		return false
	}
	if licensing.SeedCouplingRequired() && !licensing.FeatureSeedValid() {
		return false
	}
	return true
}

func reloadLicense(ctx context.Context) error {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return errLicenseWatcherUnavailable
	}
	return w.Reload(ctx)
}

func licenseWatcherState() (licensing.LicenseState, bool) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return licensing.StateExpired, false
	}
	state, _ := w.GetState()
	return state, true
}

func licenseWatcherDiagnostics() (licensing.LicenseDiagnostics, bool) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return licensing.LicenseDiagnostics{State: licensing.StateExpired}, false
	}
	state, claims := w.GetState()
	return licensing.BuildLicenseDiagnostics(claims, state, time.Now()), true
}

func licenseDeploymentLimits() (licensing.Limits, licensing.LicenseState, bool) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return licensing.Limits{}, licensing.StateExpired, false
	}
	state, claims := w.GetState()
	if claims == nil {
		return licensing.Limits{}, state, true
	}
	return claims.Limits, state, true
}

func activeActivationLicenseKey() string {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return ""
	}
	_, claims := w.GetState()
	if claims == nil {
		return ""
	}
	return licensing.ActivationLicenseKey(claims)
}

func licenseFeatureAllowed(featureKey string) (allowed bool, planCode string) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return true, ""
	}
	state, claims := w.GetState()
	if claims == nil {
		return state == licensing.StateActive || state == licensing.StateGrace || state == licensing.StateOfflineWarn, ""
	}
	planCode = claims.Plan
	ent := licensing.Entitlements{Features: claims.Features}
	switch featureKey {
	case "openrtb":
		return licensing.OpenRTBAllowed(state, ent), planCode
	case "fraud_dispute_evidence":
		return licensing.FraudDisputeEvidenceAllowed(state, ent), planCode
	default:
		return true, planCode
	}
}

func writeLicenseFeatureRequired(w http.ResponseWriter, featureKey, planCode string) {
	httpresponse.JSON(w, http.StatusForbidden, licensingadmin.LicenseFeatureRequiredBody{
		Error:       "feature_required",
		FeatureKey:  featureKey,
		PlanCode:    planCode,
		FeatureGate: featureKey,
	})
}

func requireLicenseFeature(w http.ResponseWriter, featureKey string) bool {
	allowed, planCode := licenseFeatureAllowed(featureKey)
	if allowed {
		return true
	}
	writeLicenseFeatureRequired(w, featureKey, planCode)
	return false
}

var (
	ErrCustomerNotFound                   = errors.New("customer not found")
	ErrPaymentTopupNotFound               = errors.New("payment topup not found")
	ErrRefundExceedsTopup                 = errors.New("refund exceeds settled topup")
	ErrChargebackExceedsTopup             = errors.New("chargeback exceeds settled topup")
	ErrChargebackReversalExceedsWithdrawn = errors.New("chargeback reversal exceeds withdrawn amount")

	ErrCampaignNotFound              = campaign.ErrCampaignNotFound
	ErrFraudDecisionNotFound         = errors.New("fraud decision not found")
	ErrBrandNotFound                 = campaign.ErrBrandNotFound
	ErrBrandBelongsToAnotherCustomer = campaign.ErrBrandBelongsToAnotherCustomer
	ErrCreativeNotFound              = errors.New("creative not found")
	ErrTemplateNotFound              = campaign.ErrTemplateNotFound
	ErrTeamMemberNotFound            = errors.New("team member not found")

	ErrInsufficientBalance              = campaign.ErrInsufficientBalance
	ErrTemplateBelongsToAnotherCustomer = campaign.ErrTemplateBelongsToAnotherCustomer
	ErrCampaignCannotBePaused           = campaign.ErrCampaignCannotBePaused
	ErrCampaignNotPaused                = campaign.ErrCampaignNotPaused
	ErrCampaignOutsideSchedule          = campaign.ErrCampaignOutsideSchedule
	ErrCampaignRevisionConflict         = campaign.ErrCampaignRevisionConflict
	ErrInvalidPacingMode                = campaign.ErrInvalidPacingMode
	ErrWeightMustBePositive             = errors.New("weight must be positive")
	ErrCreativeStatusInvalid            = errors.New("status must be ACTIVE or PAUSED")
	ErrIncompleteIdempotency            = campaign.ErrIncompleteIdempotency
	ErrUnsupportedGranularity           = campaign.ErrUnsupportedGranularity
	ErrInvalidTimeRange                 = campaign.ErrInvalidTimeRange
	ErrInvalidServiceFilter             = errors.New("invalid service filter")

	ErrSelfServeActiveCampaignLimit = platformadmin.ErrSelfServeActiveCampaignLimit
	ErrSelfServeDailyCreateLimit    = platformadmin.ErrSelfServeDailyCreateLimit
	ErrSelfServeBudgetOutOfRange    = platformadmin.ErrSelfServeBudgetOutOfRange
	ErrDeploymentCampaignLimit      = errors.New("deployment active campaign limit reached for license tier")
	ErrDeploymentTenantLimit        = billingadmin.ErrDeploymentTenantLimit
	ErrForbidden                    = billingadmin.ErrForbidden

	ErrFeedbackInvalidType  = platformadmin.ErrFeedbackInvalidType
	ErrFeedbackInvalidEmail = platformadmin.ErrFeedbackInvalidEmail
	ErrFeedbackEmptyMessage = platformadmin.ErrFeedbackEmptyMessage
)

func mapNotFound(err, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}

func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func errValidation(msg string) error { return validationError(msg) }

var errForbidden = &forbiddenError{}

type forbiddenError struct{}

func (e *forbiddenError) Error() string { return "forbidden" }
func mapServiceError(err error) (status int, code, message string) {
	if err == nil {
		return http.StatusOK, "", ""
	}
	if errors.Is(err, errForbidden) {
		return http.StatusForbidden, "FORBIDDEN", "forbidden"
	}
	if errors.Is(err, platformadmin.ErrInstallTokenInvalid) {
		return http.StatusUnauthorized, "UNAUTHORIZED", platformadmin.ErrInstallTokenInvalid.Error()
	}
	if errors.Is(err, platformadmin.ErrDeploymentAlreadyClaimed) {
		return http.StatusConflict, "CONFLICT", platformadmin.ErrDeploymentAlreadyClaimed.Error()
	}
	if errors.Is(err, platformadmin.ErrInviteInvalid) {
		return http.StatusBadRequest, "BAD_REQUEST", platformadmin.ErrInviteInvalid.Error()
	}

	if errors.Is(err, ErrSelfServeActiveCampaignLimit) || errors.Is(err, ErrSelfServeDailyCreateLimit) || errors.Is(err, ErrDeploymentCampaignLimit) || errors.Is(err, ErrDeploymentTenantLimit) {
		return http.StatusTooManyRequests, "LIMIT_EXCEEDED", err.Error()
	}

	var q invalidQueryError
	if errors.As(err, &q) {
		return http.StatusBadRequest, "BAD_REQUEST", string(q)
	}

	var ve validationError
	if errors.As(err, &ve) {
		return http.StatusBadRequest, "BAD_REQUEST", string(ve)
	}

	if errors.Is(err, ErrBudgetApprovalRequired) {
		return http.StatusConflict, "APPROVAL_REQUIRED", err.Error()
	}

	if errors.Is(err, ErrBudgetApprovalAutoDenied) {
		return http.StatusConflict, "APPROVAL_AUTO_DENIED", err.Error()
	}

	if errors.Is(err, dashboardadmin.ErrPublisherScopeRequired) {
		return http.StatusForbidden, "FORBIDDEN", err.Error()
	}

	if isNotFoundError(err) {
		return http.StatusNotFound, "NOT_FOUND", "resource not found"
	}

	if isConflictError(err) {
		return http.StatusConflict, "CONFLICT", conflictMessage(err)
	}

	if errors.Is(err, ErrCampaignRevisionConflict) {
		return http.StatusConflict, "CONFLICT", ErrCampaignRevisionConflict.Error()
	}

	if errors.Is(err, supply.ErrSellersJSONInvalid) {
		return http.StatusServiceUnavailable, "SUPPLY_INVALID", supply.ErrSellersJSONInvalid.Error()
	}

	if msg, ok := badRequestMessage(err); ok {
		return http.StatusBadRequest, "BAD_REQUEST", msg
	}

	return http.StatusInternalServerError, "INTERNAL_ERROR", "internal error"
}

func isNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) ||
		errors.Is(err, ErrCustomerNotFound) ||
		errors.Is(err, ErrPaymentTopupNotFound) ||
		errors.Is(err, ErrCampaignNotFound) ||
		errors.Is(err, ErrFraudDecisionNotFound) ||
		errors.Is(err, ErrBrandNotFound) ||
		errors.Is(err, ErrCreativeNotFound) ||
		errors.Is(err, ErrTemplateNotFound) ||
		errors.Is(err, ErrTeamMemberNotFound) ||
		errors.Is(err, rtbadmin.ErrRtbDealNotFound) ||
		errors.Is(err, rtbadmin.ErrDealCustomerMissing) ||
		errors.Is(err, supply.ErrSellerNotFound) ||
		errors.Is(err, supply.ErrAdsTxtEntryNotFound) ||
		errors.Is(err, domain.ErrSlotMapVersionNotFound) ||
		errors.Is(err, opsadmin.ErrDLQEntryNotFound)
}

func isConflictError(err error) bool {
	return errors.Is(err, shardadmin.ErrSlotMigrationNotReady) ||
		errors.Is(err, domain.ErrSlotMapAlreadyActive) ||
		errors.Is(err, platformadmin.ErrConfigBootstrapped) ||
		errors.Is(err, ErrCampaignRevisionConflict)
}

func conflictMessage(err error) string {
	switch {
	case errors.Is(err, shardadmin.ErrSlotMigrationNotReady):
		return shardadmin.ErrSlotMigrationNotReady.Error()
	case errors.Is(err, domain.ErrSlotMapAlreadyActive):
		return domain.ErrSlotMapAlreadyActive.Error()
	default:
		return "conflict"
	}
}

func badRequestMessage(err error) (string, bool) {
	switch {
	case errors.Is(err, licensingadmin.ErrEulaVersionMismatch):
		return licensingadmin.ErrEulaVersionMismatch.Error(), true
	case errors.Is(err, ErrFeedbackInvalidType),
		errors.Is(err, ErrFeedbackInvalidEmail),
		errors.Is(err, ErrFeedbackEmptyMessage):
		return err.Error(), true
	case errors.Is(err, ErrInsufficientBalance):
		return ErrInsufficientBalance.Error(), true
	case errors.Is(err, ErrSelfServeActiveCampaignLimit),
		errors.Is(err, ErrSelfServeDailyCreateLimit),
		errors.Is(err, ErrSelfServeBudgetOutOfRange):
		return err.Error(), true
	case errors.Is(err, ErrBrandBelongsToAnotherCustomer),
		errors.Is(err, ErrTemplateBelongsToAnotherCustomer),
		errors.Is(err, ErrCampaignCannotBePaused),
		errors.Is(err, ErrCampaignNotPaused),
		errors.Is(err, ErrCampaignOutsideSchedule),
		errors.Is(err, ErrInvalidPacingMode),
		errors.Is(err, ErrWeightMustBePositive),
		errors.Is(err, ErrCreativeStatusInvalid),
		errors.Is(err, ErrIncompleteIdempotency),
		errors.Is(err, ErrUnsupportedGranularity),
		errors.Is(err, ErrInvalidTimeRange),
		errors.Is(err, ErrInvalidServiceFilter),
		errors.Is(err, rtbadmin.ErrInvalidDealPacing),
		errors.Is(err, rtbadmin.ErrDuplicateDealID),
		errors.Is(err, rtbadmin.ErrInvalidDealSeats),
		errors.Is(err, supply.ErrInvalidSellerType),
		errors.Is(err, supply.ErrInvalidRelationship),
		errors.Is(err, supply.ErrChainTooLong),
		errors.Is(err, ErrRefundExceedsTopup),
		errors.Is(err, ErrChargebackExceedsTopup),
		errors.Is(err, ErrChargebackReversalExceedsWithdrawn),
		errors.Is(err, billingadmin.ErrExportLimit),
		errors.Is(err, domain.ErrSlotMapIncomplete),
		errors.Is(err, domain.ErrSlotMapInvalidSlot),
		errors.Is(err, domain.ErrSlotMapInvalidShard),
		errors.Is(err, platformadmin.ErrConfigNotBootstrapped):
		return err.Error(), true
	default:
		return "", false
	}
}

func writeServiceError(w http.ResponseWriter, err error, logAttrs ...any) {
	status, code, message := mapServiceError(err)
	if status >= http.StatusInternalServerError {
		attrs := append([]any{slog.Any("err", err)}, logAttrs...)
		slog.Error("management request failed", attrs...)
	}
	httpresponse.Error(w, status, code, message)
}
