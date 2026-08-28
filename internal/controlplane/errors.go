package controlplane

import (
	"ad-event-processor/internal/campaign"
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

	ErrSelfServeActiveCampaignLimit = errors.New("self-serve active campaign limit reached")
	ErrSelfServeDailyCreateLimit    = errors.New("self-serve daily campaign create limit reached")
	ErrSelfServeBudgetOutOfRange    = errors.New("self-serve budget out of allowed range")
	ErrDeploymentCampaignLimit      = errors.New("deployment active campaign limit reached for license tier")
	ErrDeploymentTenantLimit        = errors.New("deployment tenant limit reached for license tier")
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
