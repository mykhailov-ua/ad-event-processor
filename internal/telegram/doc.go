// Package telegram owns Telegram bot admin HTTP, webhook ingest, Mini App initData validation, and deeplink mint.
//
// Role:
//   - HTTP under /api/v1/telegram/* (bots, deeplinks, validate, clicks, webhook).
//   - Service stores bot tokens and webhook secrets; initdata.go validates Telegram signed payloads for /tg/* hot path glue.
//
// Topology:
//   - Wired via telegram_bridge.go; webhook ack must stay under 500ms (webhook_fault_test).
//   - Click mint and validate endpoints called from tracker edge and Mini App flows.
//
// Invariants:
//   - Webhook requires X-Telegram-Bot-Api-Secret-Token match per bot row.
//   - Deeplink tokens single-use or TTL bounded per store rules.
//   - Bot token never returned in list responses (masked DTO fields).
//
// Forbidden:
//   - Blocking CH queries on webhook handler path.
//   - Import internal/controlplane from telegram package.
//
// Verify:
//
//	go test ./internal/telegram/ -short -count=1
//	go test ./internal/telegram/ -short -run TestInitData -count=1
package telegram
