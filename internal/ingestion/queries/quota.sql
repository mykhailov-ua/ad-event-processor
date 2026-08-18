SELECT budget_limit, current_spend
FROM campaigns
WHERE id = $1
FOR UPDATE;

SELECT shard_id, campaign_id, reserved_amount, chunk_size, updated_at
FROM campaign_quotas
WHERE shard_id = $1 AND campaign_id = $2
FOR UPDATE;

INSERT INTO campaign_quotas (shard_id, campaign_id, reserved_amount, chunk_size)
VALUES ($1, $2, $3, $4);

UPDATE campaign_quotas
SET reserved_amount = reserved_amount + $3,
    chunk_size = $4,
    updated_at = NOW()
WHERE shard_id = $1 AND campaign_id = $2;

SELECT shard_id, campaign_id, reserved_amount, chunk_size, updated_at
FROM campaign_quotas
WHERE shard_id = $1 AND campaign_id = $2;

UPDATE campaign_quotas
SET reserved_amount = GREATEST(0, reserved_amount - $3),
    updated_at = NOW()
WHERE shard_id = $1 AND campaign_id = $2;

SELECT COALESCE(reserved_amount, 0)::bigint AS reserved_amount
FROM campaign_quotas
WHERE campaign_id = $1;

SELECT shard_id, campaign_id, reserved_amount, chunk_size, updated_at
FROM campaign_quotas
WHERE campaign_id = ANY($1::uuid[]) AND reserved_amount > 0;
