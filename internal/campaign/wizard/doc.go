// Package wizard stores multi-step campaign creation sessions in Postgres.
//
// Role:
//   - HTTP: GET/POST /api/v1/campaigns/wizard/session, GET /api/v1/campaigns/onboarding-templates.
//   - WizardStore persists draft JSON; publish flows call WizardHost (controlplane campaign_wizard_bridge).
//
// Invariants:
//   - Session writes require campaigns:write; reads use campaigns:read or masked variant.
//   - Published campaigns must pass EvaluateCampaignPublish before commit.
//
// Verify:
//
//	go test ./internal/campaign/ -short -run Wizard -count=1
package wizard
