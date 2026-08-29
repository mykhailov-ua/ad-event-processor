
-- name: InsertOpsMetricSample :exec
INSERT INTO ops.metric_samples (name, labels_hash, ts, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name, labels_hash, ts) DO UPDATE SET value = EXCLUDED.value;

-- name: DeleteExpiredOpsMetricSamples :execrows
DELETE FROM ops.metric_samples
WHERE ts < $1;

-- name: ListOpsMetricSamplesWindow :many
SELECT name, labels_hash, ts, value
FROM ops.metric_samples
WHERE ts >= $1
  AND ts < $2
  AND ($3::text = '' OR name = $3)
ORDER BY ts ASC, name, labels_hash;

-- name: ListOpsMetricSamplesDownsampled :many
SELECT
    name,
    labels_hash,
    to_timestamp(floor(extract(epoch FROM ts) / $4::double precision) * $4::double precision) AS ts,
    avg(value)::double precision AS value
FROM ops.metric_samples
WHERE ts >= $1
  AND ts < $2
  AND ($3::text = '' OR name = $3)
GROUP BY name, labels_hash, floor(extract(epoch FROM ts) / $4::double precision)
ORDER BY ts ASC, name, labels_hash;

-- name: GetLatestOpsMetricSample :one
SELECT name, labels_hash, ts, value
FROM ops.metric_samples
WHERE name = $1
  AND labels_hash = $2
ORDER BY ts DESC
LIMIT 1;
