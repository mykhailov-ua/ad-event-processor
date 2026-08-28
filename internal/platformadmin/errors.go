package platformadmin

import "errors"

var (
	ErrConfigBootstrapped           = errors.New("platform config already bootstrapped")
	ErrConfigNotBootstrapped        = errors.New("platform config not bootstrapped")
	ErrInstallTokenInvalid          = errors.New("invalid install token")
	ErrSelfServeActiveCampaignLimit = errors.New("self-serve active campaign limit reached")
	ErrSelfServeDailyCreateLimit    = errors.New("self-serve daily campaign create limit reached")
	ErrSelfServeBudgetOutOfRange    = errors.New("self-serve budget out of allowed range")
	ErrFeedbackInvalidType          = errors.New("invalid feedback type")
	ErrFeedbackInvalidEmail         = errors.New("invalid contact email")
	ErrFeedbackEmptyMessage         = errors.New("message is required")
)

func errPlatformServiceUnavailable() error {
	return errors.New("service unavailable")
}
