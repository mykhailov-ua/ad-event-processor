-- name: ListOutboundPostbacksByCampaign :many
SELECT id, campaign_id, name, priority, enabled, provider, url_template, api_token_encrypted,
       target_event, trigger_kind, trigger_value, test_event_code, signing_secret_encrypted,
       sample_percent, delay_seconds, created_at, updated_at
FROM campaign_outbound_postbacks
WHERE campaign_id = $1
ORDER BY priority ASC, created_at ASC;

-- name: ListOutboundPostbacksByCampaignIDs :many
SELECT id, campaign_id, name, priority, enabled, provider, url_template, api_token_encrypted,
       target_event, trigger_kind, trigger_value, test_event_code, signing_secret_encrypted,
       sample_percent, delay_seconds, created_at, updated_at
FROM campaign_outbound_postbacks
WHERE campaign_id = ANY($1::uuid[])
ORDER BY campaign_id, priority ASC, created_at ASC;

-- name: DeleteOutboundPostbacksByCampaign :exec
DELETE FROM campaign_outbound_postbacks WHERE campaign_id = $1;

-- name: InsertOutboundPostback :one
INSERT INTO campaign_outbound_postbacks (
    id, campaign_id, name, priority, enabled, provider, url_template, api_token_encrypted,
    target_event, trigger_kind, trigger_value, test_event_code, signing_secret_encrypted,
    sample_percent, delay_seconds
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
RETURNING id, campaign_id, name, priority, enabled, provider, url_template, api_token_encrypted,
          target_event, trigger_kind, trigger_value, test_event_code, signing_secret_encrypted,
          sample_percent, delay_seconds, created_at, updated_at;

-- name: GetOutboundPostback :one
SELECT id, campaign_id, name, priority, enabled, provider, url_template, api_token_encrypted,
       target_event, trigger_kind, trigger_value, test_event_code, signing_secret_encrypted,
       sample_percent, delay_seconds, created_at, updated_at
FROM campaign_outbound_postbacks
WHERE id = $1 AND campaign_id = $2;

-- name: UpdateOutboundPostback :one
UPDATE campaign_outbound_postbacks
SET name = $3,
    priority = $4,
    enabled = $5,
    provider = $6,
    url_template = $7,
    api_token_encrypted = $8,
    target_event = $9,
    trigger_kind = $10,
    trigger_value = $11,
    test_event_code = $12,
    signing_secret_encrypted = $13,
    sample_percent = $14,
    delay_seconds = $15,
    updated_at = NOW()
WHERE id = $1 AND campaign_id = $2
RETURNING id, campaign_id, name, priority, enabled, provider, url_template, api_token_encrypted,
          target_event, trigger_kind, trigger_value, test_event_code, signing_secret_encrypted,
          sample_percent, delay_seconds, created_at, updated_at;
