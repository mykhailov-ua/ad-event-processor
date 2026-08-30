// Package wizard implements multi-step campaign creation HTTP handlers and Postgres session storage.
//
// Role:
//   - HTTP: GET /api/v1/campaigns/onboarding-templates; GET/POST /api/v1/campaigns/wizard/session.
//   - WizardStore (handlers.go): PG-backed draft JSON, step validation, commit via import/export bundle.
//   - register.go init hooks campaign.SetWizardRouteRegistrar; blank import from controlplane/register.go.
//   - WizardHost port: customer access, tracking domain, integration schema, PublishCampaign on commit.
//
// Topology:
//   - Routes register on campaign.CampaignsHTTPHandlers; Service implements WizardHost in campaign_wizard_bridge.go.
//   - WizardStore lazy-created on Service.WizardStore(); handlers call Campaigns interface methods on Service.
//   - Onboarding templates loaded from embedded YAML via campaign.ListOnboardingTemplates / ApplyOnboardingTemplate.
//   - Commit builds campaign.CampaignExportBundle and imports through campaign/importexport; optional publish
//     delegates to WizardHost.PublishCampaign (publish gate in controlplane, not inline SQL here).
//
// Invariants:
//   - Wizard steps (order): traffic_source, integration_template, flow_skeleton, budget, review.
//   - POST session requires campaigns:write; GET session and templates allow campaigns:read or campaigns:read:masked.
//   - Commit requires complete step payload; incomplete commit returns validation error before PG import.
//   - Idempotency key on commit; publish=false creates draft campaign, publish=true runs PublishCampaign after import.
//   - Session DTO omits integration secrets on GET (controlplane holdout TestCampaignWizardSessionGET_omitsSecrets).
//
// Forbidden:
//   - Direct Redis or outbox writes; publish side effects flow through WizardHost / controlplane Service.
//   - Tracker ingest or filter chain imports.
//
// Verify:
//
//	go list -e ./internal/campaign/wizard/
//	go test ./internal/controlplane/ -short -run TestCampaignWizard -count=1
//	go test ./internal/controlplane/ -short -run TestCampaignWizardSessionGET_omitsSecrets -count=1
package wizard
