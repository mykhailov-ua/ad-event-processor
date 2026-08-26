package openapivalidate

// RequestValidationOperationIDs lists self-serve write operations validated when
// OPENAPI_REQUEST_VALIDATION=1. Extend deliberately; never enable on hot-path binaries.
var RequestValidationOperationIDs = map[string]struct{}{
	"selfserveCreateCampaign":      {},
	"selfservePauseCampaign":       {},
	"selfserveResumeCampaign":      {},
	"selfserveCreatePaymentIntent": {},
	"selfserveCreateApiKey":        {},
}
