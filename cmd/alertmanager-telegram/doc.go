// Package main is the Alertmanager webhook to Telegram bridge.
//
// Role:
//   - Accept POST /alerts with Alertmanager v2 webhook JSON.
//   - Format summary, severity, and description into HTML Telegram messages.
//   - POST to api.telegram.org when TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are set.
//   - Dry-run log only when token or chat id is missing or placeholder.
//
// Topology:
//   - Standalone HTTP sidecar; default listen PROXY_PORT=8222.
//   - Uses pkg/lifecycle for signal handling and HTTP server timeouts.
//   - No Postgres, Redis, or tracker wiring.
//
// Invariants:
//   - Always returns 200 {"status":"ok"} after parsing valid JSON (per-alert send errors are logged).
//   - Telegram API calls use a 10s request timeout.
//
// Forbidden:
//   - Not on tracker ingest path; no filter or stream side effects.
//   - Do not treat dry-run mode as delivered alerts.
//
// Verify:
//
//	go list -e ./cmd/alertmanager-telegram/
//
// Manual smoke (requires TELEGRAM_BOT_TOKEN): curl -X POST http://127.0.0.1:8222/alerts -d '{"status":"firing","alerts":[]}'
package main
