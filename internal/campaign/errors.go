package campaign

import (
	"errors"
	"fmt"
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
	ErrBrandNotFound                    = errors.New("brand not found")
	ErrBrandBelongsToAnotherCustomer    = errors.New("brand belongs to another customer")
	ErrTemplateNotFound                 = errors.New("template not found")
	ErrTemplateBelongsToAnotherCustomer = errors.New("template belongs to another customer")
)

func errValidation(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}
