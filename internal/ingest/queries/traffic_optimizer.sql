-- name: ListTrafficOptimizerRulesByCustomer :many
SELECT * FROM traffic_optimizer_rules
WHERE customer_id = $1
ORDER BY created_at DESC;

-- name: GetTrafficOptimizerRule :one
SELECT * FROM traffic_optimizer_rules WHERE id = $1;

-- name: ListEnabledTrafficOptimizerRules :many
SELECT * FROM traffic_optimizer_rules WHERE enabled = true ORDER BY customer_id, created_at;

-- name: InsertTrafficOptimizerRule :one
INSERT INTO traffic_optimizer_rules (
    customer_id, campaign_id, flow_id, brand_id, name, scope, objective, algorithm,
    lookback_minutes, min_clicks, min_conversions, min_spend_micro,
    eval_interval_minutes, cooldown_minutes, max_weight_delta_pct,
    preset_key, preset_parameters, enabled
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
RETURNING *;

-- name: UpdateTrafficOptimizerRule :one
UPDATE traffic_optimizer_rules
SET customer_id = $2,
    campaign_id = $3,
    flow_id = $4,
    brand_id = $5,
    name = $6,
    scope = $7,
    objective = $8,
    algorithm = $9,
    lookback_minutes = $10,
    min_clicks = $11,
    min_conversions = $12,
    min_spend_micro = $13,
    eval_interval_minutes = $14,
    cooldown_minutes = $15,
    max_weight_delta_pct = $16,
    preset_key = $17,
    preset_parameters = $18,
    enabled = $19,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTrafficOptimizerRule :execrows
DELETE FROM traffic_optimizer_rules WHERE id = $1;

-- name: UpdateTrafficOptimizerRuleLastEvaluated :exec
UPDATE traffic_optimizer_rules
SET last_evaluated_at = $2, updated_at = NOW()
WHERE id = $1;

-- name: InsertTrafficOptimizerFire :execrows
INSERT INTO traffic_optimizer_fires (
    rule_id, action_hash, campaign_id, flow_id, brand_id, payload
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING;

-- name: GetTrafficOptimizerRuleLastFiredAt :one
SELECT COALESCE(MAX(fired_at), TIMESTAMPTZ 'epoch') AS last_fired_at
FROM traffic_optimizer_fires
WHERE rule_id = $1;
