package trialregistry

import "errors"

var (
	ErrTrialTelegramUsed = errors.New("trial telegram anchor already used")
	ErrTrialHWIDUsed     = errors.New("trial hwid anchor already used")
	ErrTrialWalletUsed   = errors.New("trial usdt wallet anchor already used")
	ErrForceNotAllowed   = errors.New("trial force override requires VENDOR_TRIAL_FORCE=1")
	ErrForceReason       = errors.New("trial force override requires non-empty --force-reason")
	ErrPendingNotFound   = errors.New("pending trial request not found")
	ErrPendingNotOpen    = errors.New("pending trial request is not open")
)
