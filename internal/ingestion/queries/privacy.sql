INSERT INTO consent_events (user_id_hash, purposes, source)
VALUES ($1, $2, $3);

INSERT INTO user_consent_state (user_id_hash, ad_storage, analytics_storage, purposes, updated_at)
VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
ON CONFLICT (user_id_hash) DO UPDATE SET
    ad_storage = EXCLUDED.ad_storage,
    analytics_storage = EXCLUDED.analytics_storage,
    purposes = EXCLUDED.purposes,
    updated_at = CURRENT_TIMESTAMP;

SELECT * FROM user_consent_state WHERE user_id_hash = $1;

DELETE FROM user_consent_state WHERE user_id_hash = $1;

DELETE FROM consent_events WHERE created_at < $1;

DELETE FROM consent_events WHERE user_id_hash = $1;


UPDATE campaigns
SET require_consent_purposes = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;


INSERT INTO privacy_erasure_requests (id, user_id_hash, subject_user_id, status)
VALUES ($1, $2, $3, 'PENDING')
RETURNING *;

SELECT * FROM privacy_erasure_requests WHERE id = $1;

SELECT * FROM privacy_erasure_requests WHERE id = $1 FOR UPDATE;

SELECT * FROM privacy_erasure_requests
WHERE status = $1
ORDER BY updated_at ASC
LIMIT $2;

UPDATE privacy_erasure_requests
SET status = $2,
    last_error = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

UPDATE privacy_erasure_requests
SET subject_user_id = '',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

UPDATE events SET user_id = 'erased', ip_address = '0.0.0.0'
WHERE user_id = $1;
