package controlplane

import (
	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/platformadmin"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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
