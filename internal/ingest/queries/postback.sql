-- name: GetPostbackConfig :one
SELECT * FROM postback_configs WHERE campaign_id = $1;

-- name: ListPostbackConfigsByCampaignIDs :many
SELECT * FROM postback_configs WHERE campaign_id = ANY($1::uuid[]);

-- name: UpsertPostbackConfig :exec
INSERT INTO postback_configs (campaign_id, provider, url_template, api_token_encrypted, target_event, test_event_code, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (campaign_id) DO UPDATE
SET provider = EXCLUDED.provider,
    url_template = EXCLUDED.url_template,
    api_token_encrypted = EXCLUDED.api_token_encrypted,
    target_event = EXCLUDED.target_event,
    test_event_code = EXCLUDED.test_event_code,
    updated_at = NOW();

-- name: ClonePostbackConfig :execrows
INSERT INTO postback_configs (campaign_id, provider, url_template, api_token_encrypted, target_event, test_event_code, updated_at)
SELECT $1, src.provider, src.url_template, src.api_token_encrypted, src.target_event, src.test_event_code, NOW()
FROM postback_configs src
WHERE src.campaign_id = $2;

-- name: ListPostbackConfigs :many
SELECT * FROM postback_configs ORDER BY campaign_id;

-- name: ListPostbackCampaignStatus :many
SELECT
    c.campaign_id,
    c.provider,
    d.last_success_at::timestamptz AS last_success_at,
    COALESCE(q.dlq_pending_count, 0)::bigint AS dlq_pending_count
FROM postback_configs c
LEFT JOIN (
    SELECT campaign_id, MAX(created_at) AS last_success_at
    FROM postback_dispatches
    WHERE status = 'SENT'
    GROUP BY campaign_id
) d ON d.campaign_id = c.campaign_id
LEFT JOIN (
    SELECT campaign_id, COUNT(*)::bigint AS dlq_pending_count
    FROM postback_dlq
    WHERE status = 'FAILED'
    GROUP BY campaign_id
) q ON q.campaign_id = c.campaign_id
ORDER BY c.campaign_id;

-- name: GetPostbackDispatch :one
SELECT * FROM postback_dispatches WHERE idempotency_hash = $1;

-- name: InsertPostbackDispatch :exec
INSERT INTO postback_dispatches (idempotency_hash, campaign_id, click_id, event_type, status, error_message)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertPostbackDispatchInFlight :execrows
INSERT INTO postback_dispatches (idempotency_hash, campaign_id, click_id, event_type, status)
VALUES ($1, $2, $3, $4, 'IN_FLIGHT')
ON CONFLICT (idempotency_hash) DO NOTHING;

-- name: UpdatePostbackDispatchStatus :exec
UPDATE postback_dispatches
SET status = $2,
    error_message = $3
WHERE idempotency_hash = $1
  AND status = $4;

-- name: GetPendingPostbackEventsForUpdate :many
SELECT * FROM outbox_events
WHERE event_type = 'SEND_POSTBACK'
  AND (
    status = 'PENDING'
    OR (
      status = 'PROCESSING'
      AND processing_started_at < NOW() - INTERVAL '1 second' * $2
    )
  )
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: ListPostbackDLQ :many
SELECT * FROM postback_dlq ORDER BY created_at DESC;

-- name: GetPostbackDLQ :one
SELECT * FROM postback_dlq WHERE id = $1;

-- name: InsertPostbackDLQ :one
INSERT INTO postback_dlq (outbox_event_id, campaign_id, click_id, event_type, payload, failures_count, last_error, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdatePostbackDLQ :exec
UPDATE postback_dlq
SET failures_count = $2,
    last_error = $3,
    status = $4,
    updated_at = NOW()
WHERE id = $1;
