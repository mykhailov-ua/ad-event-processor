INSERT INTO campaign_templates (
    id, customer_id, name, budget_limit, pacing_mode, daily_budget, timezone,
    freq_limit, freq_window, target_countries, brand_id, daypart_hours
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

SELECT * FROM campaign_templates WHERE id = $1;

SELECT * FROM campaign_templates
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

SELECT COUNT(*) FROM campaign_templates WHERE customer_id = $1;


UPDATE campaigns
SET start_at = $2,
    end_at = $3,
    daypart_hours = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

SELECT * FROM campaigns
WHERE deleted_at IS NULL
  AND status IN ('ACTIVE', 'PAUSED')
  AND (start_at IS NOT NULL OR end_at IS NOT NULL)
ORDER BY updated_at ASC
LIMIT $1;

SELECT * FROM campaigns
WHERE deleted_at IS NULL
  AND status IN ('ACTIVE', 'PAUSED')
  AND (start_at IS NOT NULL OR end_at IS NOT NULL)
ORDER BY updated_at ASC
LIMIT 1
FOR UPDATE SKIP LOCKED;

UPDATE campaigns
SET status = 'PAUSED',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND status IN ('ACTIVE', 'EXHAUSTED')
RETURNING *;

UPDATE campaigns
SET status = 'ACTIVE',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND status = 'PAUSED'
RETURNING *;


INSERT INTO brand_creatives (id, brand_id, name, landing_url, weight, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

SELECT * FROM brand_creatives WHERE id = $1;

SELECT * FROM brand_creatives
WHERE brand_id = $1
ORDER BY created_at ASC;

UPDATE brand_creatives
SET name = $2,
    landing_url = $3,
    weight = $4,
    status = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

DELETE FROM brand_creatives WHERE id = $1;

SELECT * FROM brand_creatives
WHERE brand_id = $1 AND status = 'ACTIVE'
ORDER BY created_at ASC;

SELECT * FROM brand_creatives
WHERE status = 'ACTIVE'
ORDER BY brand_id, created_at ASC;

SELECT DISTINCT brand_id FROM brand_creatives WHERE status = 'ACTIVE';

SELECT id FROM campaigns
WHERE brand_id = $1 AND deleted_at IS NULL AND status IN ('ACTIVE', 'PAUSED');

SELECT brand_id, id FROM campaigns c
WHERE c.deleted_at IS NULL
  AND c.status IN ('ACTIVE', 'PAUSED')
  AND EXISTS (
    SELECT 1 FROM brand_creatives bc
    WHERE bc.brand_id = c.brand_id AND bc.status = 'ACTIVE'
  )
ORDER BY c.brand_id, c.id;

UPDATE brand_creatives
SET weight = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
