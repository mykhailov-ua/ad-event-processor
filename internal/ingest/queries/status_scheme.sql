-- name: ListStatusSchemeRulesByCampaign :many
SELECT id, campaign_id, sort_order, when_status, when_goal, set_internal_status, set_goal_name,
       payout_mode, payout_micro, fire_outbound, enabled, created_at, updated_at
FROM campaign_status_scheme_rules
WHERE campaign_id = $1
ORDER BY sort_order ASC;

-- name: ListStatusSchemeRulesByCampaignIDs :many
SELECT id, campaign_id, sort_order, when_status, when_goal, set_internal_status, set_goal_name,
       payout_mode, payout_micro, fire_outbound, enabled, created_at, updated_at
FROM campaign_status_scheme_rules
WHERE campaign_id = ANY($1::uuid[])
  AND enabled = TRUE
ORDER BY campaign_id, sort_order ASC;

-- name: DeleteStatusSchemeRulesByCampaign :exec
DELETE FROM campaign_status_scheme_rules WHERE campaign_id = $1;

-- name: InsertStatusSchemeRule :one
INSERT INTO campaign_status_scheme_rules (
    id, campaign_id, sort_order, when_status, when_goal, set_internal_status, set_goal_name,
    payout_mode, payout_micro, fire_outbound, enabled
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING id, campaign_id, sort_order, when_status, when_goal, set_internal_status, set_goal_name,
          payout_mode, payout_micro, fire_outbound, enabled, created_at, updated_at;

-- name: GetStatusSchemeRule :one
SELECT id, campaign_id, sort_order, when_status, when_goal, set_internal_status, set_goal_name,
       payout_mode, payout_micro, fire_outbound, enabled, created_at, updated_at
FROM campaign_status_scheme_rules
WHERE id = $1 AND campaign_id = $2;

-- name: UpdateStatusSchemeRule :one
UPDATE campaign_status_scheme_rules
SET when_status = $3,
    when_goal = $4,
    set_internal_status = $5,
    set_goal_name = $6,
    payout_mode = $7,
    payout_micro = $8,
    fire_outbound = $9,
    enabled = $10,
    updated_at = NOW()
WHERE id = $1 AND campaign_id = $2
RETURNING id, campaign_id, sort_order, when_status, when_goal, set_internal_status, set_goal_name,
          payout_mode, payout_micro, fire_outbound, enabled, created_at, updated_at;
