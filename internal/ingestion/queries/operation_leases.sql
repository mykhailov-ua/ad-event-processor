-- operation_leases.sql: M6 operation lease state machine (§10.3).

-- name: InsertOperationLease :one
INSERT INTO operation_leases (
    op_id,
    region_code,
    role,
    replica_set_id,
    attempt,
    factor_u,
    dedup_scope,
    lease_state,
    deadline_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'booked', $8
)
RETURNING *;

-- name: BookOperationLease :one
INSERT INTO operation_leases (
    op_id,
    region_code,
    role,
    replica_set_id,
    attempt,
    factor_u,
    dedup_scope,
    lease_state,
    deadline_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'booked',
    NOW() + (sqlc.arg(timeout_sec)::INT * INTERVAL '1 second')
)
RETURNING *;

-- name: GetOperationLease :one
SELECT * FROM operation_leases WHERE op_id = $1;

-- name: ListBookedOperationLeasesForNode :many
SELECT ol.op_id, ol.region_code, ol.role, ol.replica_set_id, ol.attempt,
       ol.factor_u, ol.dedup_scope, ol.lease_state, ol.executor_node_id,
       ol.fencing_epoch, ol.deadline_at, ol.renew_count, ol.completed_at, ol.created_at
FROM operation_leases ol
INNER JOIN operation_lease_replicas r ON r.op_id = ol.op_id
WHERE r.node_id = sqlc.arg(node_id)
  AND ol.lease_state = 'booked'
  AND ol.deadline_at > NOW()
ORDER BY ol.created_at ASC
LIMIT sqlc.arg(row_limit);

-- name: InsertOperationLeaseReplica :exec
INSERT INTO operation_lease_replicas (op_id, node_id)
VALUES ($1, $2)
ON CONFLICT (op_id, node_id) DO NOTHING;

-- name: CountOperationLeaseReplicas :one
SELECT COUNT(*)::INT AS replica_count
FROM operation_lease_replicas
WHERE op_id = $1;

-- name: CountOperationLeaseBookAcks :one
SELECT COUNT(*)::INT AS ack_count
FROM operation_lease_replicas
WHERE op_id = $1 AND book_ack_at IS NOT NULL;

-- RenewOperationLease enforces renew_count < max_renewals in SQL (M6.5 C6).
-- name: RenewOperationLease :one
UPDATE operation_leases
SET deadline_at = NOW() + (sqlc.arg(timeout_sec)::INT * INTERVAL '1 second'),
    renew_count = renew_count + 1
WHERE op_id = sqlc.arg(op_id)
  AND lease_state = 'executing'
  AND executor_node_id = sqlc.arg(executor_node_id)
  AND renew_count < sqlc.arg(max_renewals)
  AND deadline_at > NOW()
RETURNING *;

-- name: OperationLeaseClaimExecuting :many
SELECT ol.lease_state, ol.deadline_at
FROM operation_lease_claim_executing(
    sqlc.arg(op_id),
    sqlc.arg(node_id),
    sqlc.arg(fencing_epoch)
) AS ol(lease_state, deadline_at);

-- name: CompleteOperationLease :one
UPDATE operation_leases
SET lease_state = 'completed',
    completed_at = NOW()
WHERE op_id = $1
  AND lease_state = 'executing'
RETURNING *;

-- name: OperationLeaseExpireStale :one
SELECT operation_lease_expire_stale(sqlc.arg(expire_limit)::INT) AS expired_count;

-- name: UpsertOperationLeaseReplicaBookAck :exec
INSERT INTO operation_lease_replicas (op_id, node_id, book_ack_at)
VALUES ($1, $2, NOW())
ON CONFLICT (op_id, node_id) DO UPDATE
SET book_ack_at = EXCLUDED.book_ack_at;
