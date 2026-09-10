// Package replay auto-replays failed postback DLQ rows on a schedule.
//
// Role:
// - Polls postback_dlq for FAILED rows due for retry and re-enqueues SEND_POSTBACK outbox events.
//
// Invariants:
// - At-least-once replay; dispatch FSM reset allows retry after FAILED.
// - replay_count capped by max_replay_count per DLQ row.
//
// Verify:
// go test ./internal/postback/replay/ -short -count=1
// go test ./internal/postback/ -short -run TestPostback_DLQRetry_afterFailedDispatch_holdout -count=1
package replay
