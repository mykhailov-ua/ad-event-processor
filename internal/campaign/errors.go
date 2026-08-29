package campaign

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrCampaignNotFound                 = errors.New("campaign not found")
	ErrForbidden                        = errors.New("forbidden")
	ErrCampaignCannotBePaused           = errors.New("campaign cannot be paused")
	ErrCampaignNotPaused                = errors.New("campaign is not paused")
	ErrCampaignOutsideSchedule          = errors.New("campaign is outside scheduled delivery window")
	ErrCampaignPublishBlocked           = errors.New("campaign publish blocked")
	ErrCampaignRevisionConflict         = errors.New("campaign revision conflict")
	ErrCampaignWizardSessionNotFound    = errors.New("campaign wizard session not found")
	ErrCampaignWizardSessionExpired     = errors.New("campaign wizard session expired")
	ErrCampaignWizardIncomplete         = errors.New("campaign wizard session incomplete")
	ErrValidation                       = errors.New("validation error")
	ErrInvalidPacingMode                = errors.New("invalid pacing mode")
	ErrUnsupportedGranularity           = errors.New("unsupported granularity")
	ErrInvalidTimeRange                 = errors.New("invalid time range")
	ErrPostgresGateRejected             = errors.New("postgres gate rejected")
	ErrBrandNotFound                    = errors.New("brand not found")
	ErrBrandBelongsToAnotherCustomer    = errors.New("brand belongs to another customer")
	ErrTemplateNotFound                 = errors.New("template not found")
	ErrTemplateBelongsToAnotherCustomer = errors.New("template belongs to another customer")
	ErrCustomerNotFound                 = errors.New("customer not found")
	ErrInsufficientBalance              = errors.New("insufficient balance")
	ErrIncompleteIdempotency            = errors.New("incomplete idempotency")
)

func errValidation(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}

func errServiceUnavailable() error {
	return errValidation("service unavailable")
}

func ErrServiceUnavailable() error {
	return errServiceUnavailable()
}

func ErrValidationf(msg string) error {
	return errValidation(msg)
}

func IsPgUniqueViolation(err error) bool {
	return isPgUniqueViolation(err)
}

func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func MapCampaignStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCampaignNotFound
	}
	return err
}

func mapCampaignStoreError(err error) error {
	return MapCampaignStoreError(err)
}

func MapCampaignNotFound(err, notFound error) error {
	return mapCampaignNotFound(err, notFound)
}

func mapCampaignNotFound(err, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}
