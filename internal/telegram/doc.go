// Package telegram owns Telegram bot admin HTTP, webhook ingest, Mini App initData validation, deeplinks, and CH-backed reports.
//
// Role:
//   - HTTP under /api/v1/telegram/* (validate, clicks mint, webhook, bots, deeplink-tokens, postbacks).
//   - Reports under /api/v1/reports/telegram/* (summary, funnel, bots, premium, fraud).
//   - Service stores bot tokens and webhook secrets; initdata.go validates signed Telegram WebApp payloads.
//   - clickhouse_writer.go inserts tg event rows; postback_relay.go relays outbound postbacks with timeouts.
//   - outbox_event.go handles deeplink / welcome side effects from control outbox.
//
// Topology:
//   - Wired via telegram_bridge.go; Host provides PG pool, per-campaign Redis, and ClickHouse read/write conns.
//   - validate and clicks endpoints serve tracker /tg/* and Mini App flows; webhook ack must stay under 500ms.
//
// Invariants:
//   - Webhook requires X-Telegram-Bot-Api-Secret-Token match per bot row.
//   - Deeplink tokens TTL-bound in store; expired tokens rejected on read.
//   - BotDTO includes bot_token and secret_token on list/get for campaigns:read holders (restrict RBAC).
//   - CH report handlers are not on webhook path; webhook handler avoids blocking CH queries.
//
// Forbidden:
//   - Import internal/controlplane from telegram package.
//   - Blocking ClickHouse queries on webhook handler path.
//
// Verify:
//
//	go test ./internal/telegram/ -short -count=1
//	go test ./internal/telegram/ -short -run TestValidateInitData -count=1
//	go test ./internal/telegram/ -short -run TestFault_TelegramWebhook -count=1
package telegram
