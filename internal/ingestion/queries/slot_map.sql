SELECT active_version, routing_epoch, updated_at
FROM redis_slot_map_meta
WHERE id = 1;

SELECT active_version, routing_epoch, updated_at
FROM redis_slot_map_meta
WHERE id = 1
FOR UPDATE;

UPDATE redis_slot_map_meta
SET active_version = $1,
    updated_at = NOW()
WHERE id = 1;

SELECT COALESCE(MAX(version), 0)::INT AS max_version
FROM redis_slot_map;

SELECT COUNT(*)::INT AS row_count
FROM redis_slot_map
WHERE version = $1;

SELECT version, slot, shard_id, state, updated_at
FROM redis_slot_map
WHERE version = $1
ORDER BY slot ASC;

SELECT version, slot, shard_id, state, updated_at
FROM redis_slot_map
WHERE version = $1 AND state = 'MIGRATING'
ORDER BY slot ASC;

SELECT version, slot, shard_id, state, updated_at
FROM redis_slot_map
WHERE version = $1 AND slot = $2
FOR UPDATE;

INSERT INTO redis_slot_map (version, slot, shard_id, state)
VALUES ($1, $2, $3, $4);

UPDATE redis_slot_map
SET shard_id = $3,
    state = $4,
    updated_at = NOW()
WHERE version = $1 AND slot = $2;

INSERT INTO redis_slot_map (version, slot, shard_id, state)
SELECT $2, src.slot, src.shard_id, src.state
FROM redis_slot_map AS src
WHERE src.version = $1;
