-- name: InsertTelegramBot :exec
INSERT INTO telegram_bots (
    campaign_id,
    bot_id,
    bot_token,
    webhook_url,
    secret_token,
    auth_date_ttl,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, NOW(), NOW()
);

-- name: UpdateTelegramBot :exec
UPDATE telegram_bots
SET bot_id = $2,
    bot_token = $3,
    webhook_url = $4,
    secret_token = $5,
    auth_date_ttl = $6,
    updated_at = NOW()
WHERE campaign_id = $1;

-- name: GetTelegramBot :one
SELECT campaign_id, bot_id, bot_token, webhook_url, secret_token, auth_date_ttl, created_at, updated_at
FROM telegram_bots
WHERE campaign_id = $1;

-- name: GetTelegramBotByBotID :one
SELECT campaign_id, bot_id, bot_token, webhook_url, secret_token, auth_date_ttl, created_at, updated_at
FROM telegram_bots
WHERE bot_id = $1;

-- name: ListTelegramBots :many
SELECT campaign_id, bot_id, bot_token, webhook_url, secret_token, auth_date_ttl, created_at, updated_at
FROM telegram_bots
ORDER BY created_at DESC;

-- name: InsertTelegramDeeplink :exec
INSERT INTO telegram_deeplinks (
    token,
    campaign_id,
    fbclid,
    ttclid,
    utm_source,
    utm_medium,
    utm_campaign,
    utm_term,
    utm_content,
    landing_ts,
    expires_at,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
);

-- name: GetTelegramDeeplink :one
SELECT token, campaign_id, fbclid, ttclid, utm_source, utm_medium, utm_campaign, utm_term, utm_content, landing_ts, expires_at, created_at
FROM telegram_deeplinks
WHERE token = $1;

-- name: DeleteTelegramDeeplink :exec
DELETE FROM telegram_deeplinks
WHERE token = $1;

-- name: TryClaimTelegramWebhookUpdate :exec
INSERT INTO telegram_webhook_idempotency (update_id, processed_at)
VALUES ($1, NOW());

-- name: InsertTelegramPostback :exec
INSERT INTO telegram_postbacks (
    id,
    campaign_id,
    postback_url,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, NOW(), NOW()
);

-- name: UpdateTelegramPostback :exec
UPDATE telegram_postbacks
SET postback_url = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: GetTelegramPostback :one
SELECT id, campaign_id, postback_url, created_at, updated_at
FROM telegram_postbacks
WHERE id = $1;

-- name: ListTelegramPostbacksByCampaign :many
SELECT id, campaign_id, postback_url, created_at, updated_at
FROM telegram_postbacks
WHERE campaign_id = $1
ORDER BY created_at DESC;

-- name: DeleteTelegramPostback :exec
DELETE FROM telegram_postbacks
WHERE id = $1;
