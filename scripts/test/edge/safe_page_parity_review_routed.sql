-- Optional ClickHouse template: review-routed click rate vs total clicks (7d).
-- Use when validating sandbox routing volume after safe-page parity drills.
-- Verify: clickhouse-client --query "$(cat scripts/test/edge/safe_page_parity_review_routed.sql)"

SELECT
    campaign_id,
    count() AS clicks_total,
    countIf(review_routed_event = 1) AS clicks_review_routed,
    round(clicks_review_routed / greatest(clicks_total, 1), 4) AS review_routed_rate
FROM clicks
WHERE event_time >= now() - INTERVAL 7 DAY
GROUP BY campaign_id
HAVING clicks_total >= 100
ORDER BY review_routed_rate DESC
LIMIT 50;
