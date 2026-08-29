-- +goose Up
-- +goose StatementBegin
CREATE TABLE operation_leases (
    op_id            UUID PRIMARY KEY,
    region_code      SMALLINT NOT NULL,
    role             TEXT NOT NULL,
    replica_set_id   UUID NOT NULL,
    attempt          INT NOT NULL DEFAULT 1,
    factor_u         UUID NOT NULL,
    dedup_scope      JSONB NOT NULL,
    lease_state      TEXT NOT NULL DEFAULT 'booked',
    executor_node_id TEXT,
    fencing_epoch    BIGINT NOT NULL DEFAULT 0,
    deadline_at      TIMESTAMPTZ NOT NULL,
    renew_count      INT NOT NULL DEFAULT 0,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT operation_leases_state_chk CHECK (
        lease_state IN ('booked', 'executing', 'completed', 'expired')
    ),
    CONSTRAINT operation_leases_attempt_uniq UNIQUE (replica_set_id, attempt)
);

CREATE INDEX idx_operation_leases_active
    ON operation_leases (region_code, role, lease_state, deadline_at)
    WHERE lease_state IN ('booked', 'executing');

CREATE TABLE operation_lease_replicas (
    op_id           UUID NOT NULL REFERENCES operation_leases(op_id) ON DELETE CASCADE,
    node_id         TEXT NOT NULL,
    book_ack_at     TIMESTAMPTZ,
    local_flags     SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (op_id, node_id)
);

CREATE INDEX idx_operation_lease_replicas_node
    ON operation_lease_replicas (node_id, book_ack_at DESC);

CREATE OR REPLACE FUNCTION operation_lease_claim_executing(
    p_op_id UUID,
    p_node_id TEXT,
    p_fencing_epoch BIGINT
) RETURNS TABLE(lease_state TEXT, deadline_at TIMESTAMPTZ) AS $$
BEGIN
    RETURN QUERY
    UPDATE operation_leases ol
    SET lease_state = 'executing',
        executor_node_id = p_node_id,
        fencing_epoch = p_fencing_epoch
    WHERE ol.op_id = p_op_id
      AND ol.lease_state = 'booked'
      AND ol.deadline_at > NOW()
    RETURNING ol.lease_state, ol.deadline_at;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION operation_lease_expire_stale(p_limit INT DEFAULT 500)
RETURNS INT AS $$
DECLARE
    n INT;
BEGIN
    WITH expired AS (
        SELECT op_id FROM operation_leases
        WHERE lease_state IN ('booked', 'executing')
          AND deadline_at < NOW()
        ORDER BY deadline_at ASC
        LIMIT p_limit
        FOR UPDATE SKIP LOCKED
    )
    UPDATE operation_leases ol
    SET lease_state = 'expired'
    FROM expired e
    WHERE ol.op_id = e.op_id;
    GET DIAGNOSTICS n = ROW_COUNT;
    RETURN n;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS operation_lease_expire_stale(INT);
DROP FUNCTION IF EXISTS operation_lease_claim_executing(UUID, TEXT, BIGINT);
DROP TABLE IF EXISTS operation_lease_replicas;
DROP TABLE IF EXISTS operation_leases;
-- +goose StatementEnd
