
-- +goose Up
-- +goose StatementBegin
CREATE TYPE redis_slot_state AS ENUM ('ACTIVE', 'MIGRATING', 'DRAINING');

CREATE TABLE redis_slot_map (
    version    INT NOT NULL,
    slot       SMALLINT NOT NULL CHECK (slot >= 0 AND slot <= 1023),
    shard_id   SMALLINT NOT NULL CHECK (shard_id >= 0),
    state      redis_slot_state NOT NULL DEFAULT 'ACTIVE',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (version, slot)
);

CREATE INDEX idx_redis_slot_map_version_state ON redis_slot_map (version, state);

CREATE TABLE redis_slot_map_meta (
    id             INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    active_version INT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO redis_slot_map (version, slot, shard_id, state)
SELECT 1, gs::SMALLINT, (gs % 4)::SMALLINT, 'ACTIVE'::redis_slot_state
FROM generate_series(0, 1023) AS gs;

INSERT INTO redis_slot_map_meta (id, active_version)
VALUES (1, 1);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS redis_slot_map_meta;
DROP TABLE IF EXISTS redis_slot_map;
DROP TYPE IF EXISTS redis_slot_state;
-- +goose StatementEnd
