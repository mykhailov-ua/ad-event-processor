package platformadmin

import "ad-event-processor/internal/platformadmin/domains"

type (
	DomainCloudflareClient   = domains.DomainCloudflareClient
	DomainHealthHost         = domains.DomainHealthHost
	DomainHealth             = domains.DomainHealth
	DomainHealthService      = domains.DomainHealthService
	DomainHealthDTO          = domains.DomainHealthDTO
	DomainSSLSetupResult     = domains.DomainSSLSetupResult
	DomainTLSAllowedResponse = domains.DomainTLSAllowedResponse
	ParkDomainRequest        = domains.ParkDomainRequest
	ParkDomainResponse       = domains.ParkDomainResponse
	WildcardSSLRequest       = domains.WildcardSSLRequest
	WildcardSSLResponse      = domains.WildcardSSLResponse
	DomainHealthHTTPHandlers = domains.DomainHealthHTTPHandlers
	CloudflareZone           = domains.CloudflareZone
	CloudflareAPI            = domains.CloudflareAPI
	DomainBulkRequest        = domains.DomainBulkRequest
	DomainBulkJobStatus      = domains.DomainBulkJobStatus
	DomainBulkJobRow         = domains.DomainBulkJobRow
	BurnDomainRequest        = domains.BurnDomainRequest
	BurnDomainResponse       = domains.BurnDomainResponse
)

var (
	NewDomainHealth         = domains.NewDomainHealth
	NewCloudflareClient     = domains.NewCloudflareClient
	StartDomainHealthWorker = domains.StartDomainHealthWorker
)
