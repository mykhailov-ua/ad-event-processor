// Package openapivalidate validates selected admin HTTP request bodies against the bundled OpenAPI spec.
//
// Role:
//   - NewRequestValidationMiddleware loads api/openapi/openapi.bundle.yaml via kin-openapi and validates request bodies.
//   - RequestValidationOperationIDs gates validation to five selfserve operationIds (create campaign, pause, resume, payment intent, API key).
//   - ResolveRequestValidationMiddleware wires enabled flag from config.Management.OpenAPIRequestValidation in controlplane admin_static.go.
//
// Topology:
//   - Cold path only: middleware on cmd/control admin mux before handler bodies run.
//   - Unlisted operations and router miss pass through without validation.
//
// Invariants:
//   - When enabled, schema violations return 400 BAD_REQUEST before the route handler runs.
//   - Enabled=false or bundle load failure when disabled logs and uses passthrough middleware.
//   - Response bodies are not validated (ExcludeResponseBody true).
//
// Forbidden:
//   - OpenAPI request validation on tracker /track or other hot-path handlers.
//
// Verify:
//
//	go test ./internal/openapivalidate/ -short -count=1
//	go test ./internal/openapivalidate/ -short -run TestOpenAPIRequestValidation -count=1
package openapivalidate
