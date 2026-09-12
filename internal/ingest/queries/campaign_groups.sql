-- name: ListCampaignGroupsByCustomer :many
SELECT id, customer_id, name, default_flow_id, created_at, updated_at, deleted_at
FROM campaign_groups
WHERE deleted_at IS NULL
  AND customer_id = $1
ORDER BY lower(name) ASC, created_at ASC;

-- name: GetCampaignGroup :one
SELECT id, customer_id, name, default_flow_id, created_at, updated_at, deleted_at
FROM campaign_groups
WHERE id = $1
  AND deleted_at IS NULL;

-- name: InsertCampaignGroup :one
INSERT INTO campaign_groups (id, customer_id, name, default_flow_id)
VALUES ($1, $2, $3, $4)
RETURNING id, customer_id, name, default_flow_id, created_at, updated_at, deleted_at;

-- name: UpdateCampaignGroup :one
UPDATE campaign_groups
SET name = $2,
    default_flow_id = $3,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, customer_id, name, default_flow_id, created_at, updated_at, deleted_at;

-- name: SoftDeleteCampaignGroup :exec
UPDATE campaign_groups
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ClearCampaignGroupMembers :exec
UPDATE campaigns
SET campaign_group_id = NULL,
    updated_at = NOW()
WHERE campaign_group_id = $1
  AND deleted_at IS NULL;

-- name: AssignCampaignsToGroup :execrows
UPDATE campaigns
SET campaign_group_id = sqlc.arg('campaign_group_id'),
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND customer_id = sqlc.arg('customer_id')
  AND id = ANY(sqlc.arg('campaign_ids')::uuid[]);

-- name: UnassignCampaignsFromGroup :execrows
UPDATE campaigns
SET campaign_group_id = NULL,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND customer_id = sqlc.arg('customer_id')
  AND campaign_group_id = sqlc.arg('campaign_group_id')
  AND id = ANY(sqlc.arg('campaign_ids')::uuid[]);

-- name: ListCampaignIDsByGroup :many
SELECT id
FROM campaigns
WHERE deleted_at IS NULL
  AND customer_id = $1
  AND campaign_group_id = $2
ORDER BY updated_at DESC;

-- name: CountCampaignsByGroup :one
SELECT COUNT(*)::bigint
FROM campaigns
WHERE deleted_at IS NULL
  AND customer_id = $1
  AND campaign_group_id = $2;
