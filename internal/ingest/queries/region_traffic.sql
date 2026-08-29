
-- name: ListActiveRegionCodes :many
SELECT code
FROM regions
WHERE active = TRUE
  AND code <> 0
ORDER BY code;

-- name: UpsertRegionTrafficDial :exec
INSERT INTO region_traffic_dial (
    region_code, score, weight, provenance, epoch_id, updated_at
) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (region_code) DO UPDATE SET
    score = EXCLUDED.score,
    weight = EXCLUDED.weight,
    provenance = EXCLUDED.provenance,
    epoch_id = EXCLUDED.epoch_id,
    updated_at = NOW();

-- name: ListRegionTrafficDial :many
SELECT region_code, score, weight, provenance, epoch_id, updated_at
FROM region_traffic_dial
ORDER BY region_code;

-- name: GetRegionTrafficDial :one
SELECT region_code, score, weight, provenance, epoch_id, updated_at
FROM region_traffic_dial
WHERE region_code = $1;
