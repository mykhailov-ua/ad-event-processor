INSERT INTO node_metric_buckets (
    node_id, region_code, role, bucket_ts, metric,
    value_p50, value_p99, value_mean, sample_count
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (node_id, bucket_ts, metric) DO UPDATE SET
    value_p50 = EXCLUDED.value_p50,
    value_p99 = EXCLUDED.value_p99,
    value_mean = EXCLUDED.value_mean,
    sample_count = EXCLUDED.sample_count;

SELECT node_id, region_code, role, bucket_ts, metric,
       value_p50, value_p99, value_mean, sample_count
FROM node_metric_buckets
WHERE region_code = $1
  AND role = $2
  AND bucket_ts >= $3
  AND bucket_ts < $4
ORDER BY bucket_ts DESC, node_id, metric;

DELETE FROM node_metric_buckets
WHERE bucket_ts < $1;

SELECT
    region_code,
    role,
    metric,
    AVG(value_p50) AS value_p50,
    MAX(value_p99) AS value_p99,
    AVG(value_mean) AS value_mean,
    SUM(sample_count)::bigint AS sample_count
FROM node_metric_buckets
WHERE bucket_ts >= $1
  AND bucket_ts < $2
GROUP BY region_code, role, metric;

INSERT INTO node_metric_daily_snapshots (
    day, region_code, role, metric,
    value_p50, value_p99, value_mean, sample_count
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (day, region_code, role, metric) DO UPDATE SET
    value_p50 = EXCLUDED.value_p50,
    value_p99 = EXCLUDED.value_p99,
    value_mean = EXCLUDED.value_mean,
    sample_count = EXCLUDED.sample_count;

SELECT day, region_code, role, metric,
       value_p50, value_p99, value_mean, sample_count
FROM node_metric_daily_snapshots
WHERE day = $1
  AND region_code = $2
  AND role = $3
  AND metric = $4;

SELECT day, region_code, role, metric,
       value_p50, value_p99, value_mean, sample_count
FROM node_metric_daily_snapshots
WHERE day = $1
  AND region_code = $2
  AND role = $3
ORDER BY metric;

SELECT node_id, region_code, role, bucket_ts, metric,
       value_p50, value_p99, value_mean, sample_count
FROM node_metric_buckets
WHERE region_code = $1
  AND role = $2
  AND bucket_ts >= $3
  AND bucket_ts < $4
ORDER BY node_id, metric, bucket_ts;

INSERT INTO node_capacity_scores (
    node_id, region_code, role, score, weight, provenance, epoch_id, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (node_id, region_code, role) DO UPDATE SET
    score = EXCLUDED.score,
    weight = EXCLUDED.weight,
    provenance = EXCLUDED.provenance,
    epoch_id = EXCLUDED.epoch_id,
    updated_at = NOW();

SELECT node_id, region_code, role, score, weight, provenance, epoch_id, updated_at
FROM node_capacity_scores
WHERE region_code = $1
  AND role = $2;

SELECT node_id, region_code, role, score, weight, provenance, epoch_id, updated_at
FROM node_capacity_scores
WHERE role = $1
ORDER BY region_code, node_id;

SELECT node_id, region_code, role, score, weight, provenance, epoch_id, updated_at
FROM node_capacity_scores
WHERE region_code = $1
ORDER BY role, node_id;
