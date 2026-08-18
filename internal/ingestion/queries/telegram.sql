INSERT INTO telegram_bots (
    campaign_id,
    bot_id,
    bot_token,
    webhook_url,
    mini_app_url,
    secret_token,
    auth_date_ttl,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
);

UPDATE telegram_bots
SET bot_id = $2,
    bot_token = $3,
    webhook_url = $4,
    mini_app_url = $5,
    secret_token = $6,
    auth_date_ttl = $7,
    updated_at = NOW()
WHERE campaign_id = $1;

SELECT campaign_id, bot_id, bot_token, webhook_url, mini_app_url, secret_token, auth_date_ttl, created_at, updated_at
FROM telegram_bots
WHERE campaign_id = $1;

SELECT campaign_id, bot_id, bot_token, webhook_url, mini_app_url, secret_token, auth_date_ttl, created_at, updated_at
FROM telegram_bots
WHERE bot_id = $1;

SELECT campaign_id, bot_id, bot_token, webhook_url, mini_app_url, secret_token, auth_date_ttl, created_at, updated_at
FROM telegram_bots
ORDER BY created_at DESC;

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

SELECT token, campaign_id, fbclid, ttclid, utm_source, utm_medium, utm_campaign, utm_term, utm_content, landing_ts, expires_at, created_at
FROM telegram_deeplinks
WHERE token = $1;

DELETE FROM telegram_deeplinks
WHERE token = $1;

INSERT INTO telegram_webhook_idempotency (update_id, processed_at)
VALUES ($1, NOW());

INSERT INTO telegram_postbacks (
    id,
    campaign_id,
    postback_url,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, NOW(), NOW()
);

UPDATE telegram_postbacks
SET postback_url = $2,
    updated_at = NOW()
WHERE id = $1;

SELECT id, campaign_id, postback_url, created_at, updated_at
FROM telegram_postbacks
WHERE id = $1;

SELECT id, campaign_id, postback_url, created_at, updated_at
FROM telegram_postbacks
WHERE campaign_id = $1
ORDER BY created_at DESC;

DELETE FROM telegram_postbacks
WHERE id = $1;
