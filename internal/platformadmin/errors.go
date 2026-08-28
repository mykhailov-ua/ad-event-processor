package platformadmin

import "errors"

var (
	ErrConfigBootstrapped    = errors.New("platform config already bootstrapped")
	ErrConfigNotBootstrapped = errors.New("platform config not bootstrapped")
	ErrInstallTokenInvalid   = errors.New("invalid install token")
)
