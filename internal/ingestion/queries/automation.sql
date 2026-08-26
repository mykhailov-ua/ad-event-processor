-- name: ListAutomationRulesByCustomer :many
SELECT * FROM automation_rules
WHERE customer_id = $1
ORDER BY created_at DESC;

-- name: GetAutomationRule :one
SELECT * FROM automation_rules WHERE id = $1;

-- name: InsertAutomationRule :one
INSERT INTO automation_rules (
    customer_id, campaign_id, name, metric, operator, threshold,
    window_minutes, group_by, actions, cooldown_minutes, enabled
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateAutomationRule :one
UPDATE automation_rules
SET name = $2,
    campaign_id = $3,
    metric = $4,
    operator = $5,
    threshold = $6,
    window_minutes = $7,
    group_by = $8,
    actions = $9,
    cooldown_minutes = $10,
    enabled = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAutomationRule :execrows
DELETE FROM automation_rules WHERE id = $1;

-- name: ListEnabledAutomationRules :many
SELECT * FROM automation_rules WHERE enabled = true ORDER BY customer_id, created_at;

-- name: InsertAutomationRuleFire :execrows
INSERT INTO automation_rule_fires (
    rule_id, action_hash, campaign_id, placement_id, metric, observed_value, payload
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT DO NOTHING;

-- name: UpdateAutomationRuleLastFired :exec
UPDATE automation_rules
SET last_fired_at = $2, updated_at = NOW()
WHERE id = $1;
