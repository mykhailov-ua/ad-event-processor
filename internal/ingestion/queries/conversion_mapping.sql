-- name: ListConversionMappingsByCampaign :many
SELECT campaign_id, inbound_status, goal_name, payout_micro, created_at, updated_at
FROM campaign_conversion_mappings
WHERE campaign_id = $1
ORDER BY inbound_status;

-- name: ListConversionMappingsByCampaignIDs :many
SELECT campaign_id, inbound_status, goal_name, payout_micro, created_at, updated_at
FROM campaign_conversion_mappings
WHERE campaign_id = ANY($1::uuid[])
ORDER BY campaign_id, inbound_status;

-- name: DeleteConversionMappingsByCampaign :exec
DELETE FROM campaign_conversion_mappings WHERE campaign_id = $1;

-- name: InsertConversionMapping :exec
INSERT INTO campaign_conversion_mappings (campaign_id, inbound_status, goal_name, payout_micro)
VALUES ($1, $2, $3, $4);
