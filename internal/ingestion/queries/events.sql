
-- name: CreateCampaign :one
INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window, target_countries, brand_id, brand_fcap_key, start_at, end_at, daypart_hours, template_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING *;

-- name: GetCampaign :one
SELECT * FROM campaigns WHERE id = $1 LIMIT 1;

-- name: InsertEvent :exec
INSERT INTO events (click_id, campaign_id, user_id, event_type, payload, ip_address, user_agent, created_at, created_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (click_id, created_date) DO NOTHING;

-- name: UpdateCampaignStats :exec
INSERT INTO campaign_stats (campaign_id, date, impressions_count, clicks_count, conversions_count)
VALUES ($1, CURRENT_DATE, $2, $3, $4)
ON CONFLICT (campaign_id, date) DO UPDATE SET
    impressions_count = campaign_stats.impressions_count + EXCLUDED.impressions_count,
    clicks_count = campaign_stats.clicks_count + EXCLUDED.clicks_count,
    conversions_count = campaign_stats.conversions_count + EXCLUDED.conversions_count;

-- name: GetCampaignStats :many
SELECT * FROM campaign_stats 
WHERE campaign_id = $1 
ORDER BY date DESC;

-- name: ListCampaignIDs :many
SELECT id FROM campaigns WHERE status = 'ACTIVE';

-- name: UpdateCampaignStatsBatch :exec
INSERT INTO campaign_stats (campaign_id, date, impressions_count, clicks_count, conversions_count)
SELECT 
    val.campaign_id,
    CURRENT_DATE,
    val.impression,
    val.click,
    val.conversion
FROM (
    SELECT 
        unnest(@campaign_ids::uuid[]) as campaign_id,
        unnest(@impressions::bigint[]) as impression,
        unnest(@clicks::bigint[]) as click,
        unnest(@conversions::bigint[]) as conversion
) val
ORDER BY val.campaign_id
ON CONFLICT (campaign_id, date) DO UPDATE SET
    impressions_count = campaign_stats.impressions_count + EXCLUDED.impressions_count,
    clicks_count = campaign_stats.clicks_count + EXCLUDED.clicks_count,
    conversions_count = campaign_stats.conversions_count + EXCLUDED.conversions_count;

-- name: InsertEventsBatch :exec
WITH inserted AS (
    INSERT INTO events (click_id, campaign_id, user_id, event_type, payload, ip_address, user_agent, created_at, created_date)
    SELECT 
        unnest(@click_ids::text[]),
        unnest(@campaign_ids::uuid[]),
        unnest(@user_ids::text[]),
        unnest(@event_types::text[]),
        unnest(@payloads::jsonb[]),
        unnest(@ip_addresses::text[]),
        unnest(@user_agents::text[]),
        unnest(@created_at::timestamptz[]),
        unnest(@created_dates::date[])
    ON CONFLICT (click_id, created_date) DO NOTHING
    RETURNING campaign_id, event_type, created_date
),
stats AS (
    SELECT i.campaign_id,
           i.created_date as event_date,
           COUNT(*) FILTER (WHERE i.event_type = 'impression') as imps,
           COUNT(*) FILTER (WHERE i.event_type = 'click') as clicks,
           COUNT(*) FILTER (WHERE i.event_type = 'conversion') as convs
    FROM inserted i
    WHERE EXISTS (SELECT 1 FROM campaigns c WHERE c.id = i.campaign_id)
    GROUP BY i.campaign_id, i.created_date
    ORDER BY i.campaign_id, i.created_date
)
INSERT INTO campaign_stats (campaign_id, date, impressions_count, clicks_count, conversions_count)
SELECT campaign_id, event_date, imps, clicks, convs
FROM stats
ORDER BY campaign_id, event_date
ON CONFLICT (campaign_id, date) DO UPDATE SET
    impressions_count = campaign_stats.impressions_count + EXCLUDED.impressions_count,
    clicks_count = campaign_stats.clicks_count + EXCLUDED.clicks_count,
    conversions_count = campaign_stats.conversions_count + EXCLUDED.conversions_count;
