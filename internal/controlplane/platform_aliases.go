package controlplane

import "ad-event-processor/internal/platformadmin"

type PlatformHTTPHandlers = platformadmin.HTTPHandlers
type PlatformConfigService = platformadmin.Service
type PlatformAuthClient = platformadmin.AuthClient

var (
	ErrPlatformConfigBootstrapped    = platformadmin.ErrConfigBootstrapped
	ErrPlatformConfigNotBootstrapped = platformadmin.ErrConfigNotBootstrapped
	ErrInstallTokenInvalid           = platformadmin.ErrInstallTokenInvalid
)
