-- name: ListPlatformCampaignLinks :many
SELECT * FROM platform_campaign_links
WHERE ($1::uuid IS NULL OR campaign_id = $1)
  AND ($2::uuid IS NULL OR customer_id = $2)
ORDER BY campaign_id, network;

-- name: GetPlatformCampaignLink :one
SELECT * FROM platform_campaign_links
WHERE campaign_id = $1 AND network = $2;

-- name: UpsertPlatformCampaignLink :one
INSERT INTO platform_campaign_links (
    campaign_id, customer_id, network, external_campaign_id, account_id, updated_at
)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (campaign_id, network) DO UPDATE
SET external_campaign_id = EXCLUDED.external_campaign_id,
    account_id = COALESCE(NULLIF(EXCLUDED.account_id, ''), platform_campaign_links.account_id),
    updated_at = NOW()
RETURNING *;

-- name: DeletePlatformCampaignLink :exec
DELETE FROM platform_campaign_links WHERE campaign_id = $1 AND network = $2;

-- name: UpdatePlatformCampaignLinkStatus :exec
UPDATE platform_campaign_links
SET external_status = $3,
    external_budget_resource = $4,
    external_daily_budget_micro = $5,
    last_synced_at = NOW(),
    sync_error = $6,
    updated_at = NOW()
WHERE campaign_id = $1 AND network = $2;

-- name: ListPlatformCampaignLinksForSync :many
SELECT * FROM platform_campaign_links
ORDER BY last_synced_at NULLS FIRST, id
LIMIT $1;

-- name: InsertPlatformCampaignMutation :one
INSERT INTO platform_campaign_mutations (
    idempotency_key, campaign_id, customer_id, network, action, request_json, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetPlatformCampaignMutationByKey :one
SELECT * FROM platform_campaign_mutations WHERE idempotency_key = $1;

-- name: ListPendingPlatformCampaignMutations :many
SELECT * FROM platform_campaign_mutations
WHERE status = 'pending'
ORDER BY created_at
LIMIT $1;

-- name: CompletePlatformCampaignMutation :exec
UPDATE platform_campaign_mutations
SET status = $2,
    response_json = $3,
    error_message = $4,
    applied_at = NOW()
WHERE id = $1;
