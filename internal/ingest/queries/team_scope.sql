-- name: ListTeamsByCustomer :many
SELECT id, customer_id, name, created_at
FROM teams
WHERE customer_id = $1
ORDER BY name ASC, id ASC;

-- name: CreateTeam :one
INSERT INTO teams (id, customer_id, name)
VALUES ($1, $2, $3)
RETURNING id, customer_id, name, created_at;

-- name: GetCustomerTeamEnforceOwnership :one
SELECT team_enforce_ownership
FROM customers
WHERE id = $1;

-- name: ResetPostbackDispatchForRetry :execrows
UPDATE postback_dispatches
SET status = 'IN_FLIGHT',
    error_message = NULL
WHERE idempotency_hash = $1
  AND status = 'FAILED';

-- name: ListPostbackDLQForAutoReplay :many
SELECT *
FROM postback_dlq
WHERE status = 'FAILED'
  AND replay_count < max_replay_count
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
ORDER BY created_at ASC
LIMIT $1;

-- name: UpdatePostbackDLQReplay :exec
UPDATE postback_dlq
SET replay_count = $2,
    next_retry_at = $3,
    status = $4,
    last_error = $5,
    updated_at = NOW()
WHERE id = $1;

-- name: GetCustomerPostbackInboundAuth :one
SELECT postback_inbound_ip_allowlist, postback_inbound_secret_encrypted
FROM customers
WHERE id = $1;
