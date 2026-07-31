
WITH
    shadow AS (
        SELECT
            ip_hash,
            max(score) AS ml_score
        FROM ml_shadow_scores
        WHERE created_at >= now() - INTERVAL 24 HOUR
        GROUP BY ip_hash
    ),
    positives AS (
        SELECT DISTINCT ip_hash
        FROM fraud_events
        WHERE created_at >= now() - INTERVAL 24 HOUR
          AND fraud_reason != ''
    ),
    negatives AS (
        SELECT DISTINCT i.ip_hash
        FROM impressions AS i
        WHERE i.created_at >= now() - INTERVAL 24 HOUR
          AND i.ip_hash NOT IN (SELECT ip_hash FROM positives)
        LIMIT 10000
    ),
    labeled AS (
        SELECT s.ip_hash, s.ml_score, toUInt8(1) AS label
        FROM shadow AS s
        INNER JOIN positives AS p USING (ip_hash)
        UNION ALL
        SELECT s.ip_hash, s.ml_score, toUInt8(0) AS label
        FROM shadow AS s
        INNER JOIN negatives AS n USING (ip_hash)
    )
SELECT
    countIf(label = 1 AND ml_score >= 0.6) AS tp,
    countIf(label = 0 AND ml_score >= 0.6) AS fp,
    countIf(label = 1 AND ml_score < 0.6) AS fn,
    countIf(label = 0 AND ml_score < 0.6) AS tn,
    if(tp + fp = 0, 0, tp / (tp + fp)) AS precision,
    if(tp + fn = 0, 0, tp / (tp + fn)) AS recall
FROM labeled;
