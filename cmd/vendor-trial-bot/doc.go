// Package main runs the vendor Telegram bot for trial intake.
//
// Role:
//   - Subcommand run: long-poll Telegram getUpdates and handle /trial requests.
//   - Append pending rows to trial registry JSON via internal/trialregistry.
//   - Dry-run mode logs without writing registry or calling Telegram when --dry-run set.
//
// Topology:
//   - Vendor VPS sidecar; HTTP client to api.telegram.org only.
//   - Approval and JWT issue happen offline via cmd/trial-registry and cmd/license-issue.
//
// Defaults and limits:
//   - run --poll-timeout default 30s (Telegram long-poll).
//   - --trial-registry overrides registry path for run subcommand.
//   - VENDOR_TRIAL_BOT_TOKEN required unless --dry-run.
//   - VENDOR_TRIAL_REGISTRY registry file path (default deploy/vendor/trial_registry.json).
//
// Forbidden:
//   - Never commit bot token; not deployed on customer tracker/control hosts.
//   - Does not mutate buyer deployment license files directly.
//
// Verify:
// go test ./cmd/vendor-trial-bot/... -short -count=1
// go run ./cmd/vendor-trial-bot help
package main
