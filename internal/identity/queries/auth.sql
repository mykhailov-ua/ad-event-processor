SELECT id, email, password_hash, role, customer_id, created_at, updated_at, is_blocked, email_verified, seller_id, publisher_account_id
FROM users
WHERE email = $1;

SELECT id, email, password_hash, role, customer_id, created_at, updated_at, is_blocked, email_verified, seller_id, publisher_account_id
FROM users
WHERE id = $1;

INSERT INTO users (email, password_hash, role, customer_id, email_verified)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, role, customer_id, created_at;

UPDATE users
SET is_blocked = FALSE, updated_at = NOW()
WHERE email = $1;

UPDATE users
SET password_hash = $2, updated_at = NOW()
WHERE email = $1;

UPDATE users
SET is_blocked = TRUE, updated_at = NOW()
WHERE email = $1;

SELECT ak.id, ak.user_id, ak.name, ak.expires_at, u.role, u.customer_id
FROM api_keys ak
JOIN users u ON ak.user_id = u.id
WHERE ak.key_hash = $1 AND (ak.expires_at IS NULL OR ak.expires_at > NOW());

SELECT ak.id, ak.user_id, ak.name, ak.key_hash, ak.expires_at, u.role, u.customer_id
FROM api_keys ak
JOIN users u ON ak.user_id = u.id
WHERE ak.key_lookup = $1 AND (ak.expires_at IS NULL OR ak.expires_at > NOW());

INSERT INTO api_keys (key_hash, key_lookup, user_id, name, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, expires_at, created_at;

SELECT id, name, expires_at, created_at
FROM api_keys
WHERE user_id = $1;

INSERT INTO sessions (id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at;

SELECT id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at
FROM sessions
WHERE id = $1;

SELECT id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at
FROM sessions
WHERE refresh_token = $1;

SELECT id, user_id, refresh_token, user_agent, client_ip, is_blocked, expires_at, created_at
FROM sessions
WHERE refresh_token = $1
FOR UPDATE;

UPDATE sessions
SET is_blocked = TRUE
WHERE id = $1;

UPDATE sessions
SET is_blocked = TRUE
WHERE refresh_token = $1;

DELETE FROM sessions
WHERE expires_at < NOW() OR is_blocked = TRUE;

UPDATE users
SET email_verified = TRUE, updated_at = NOW()
WHERE id = $1;

INSERT INTO auth_audit_log (user_id, action, target_type, target_id, client_ip, user_agent, changes, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at;

SELECT id, user_id, action, target_type, target_id, client_ip, user_agent, changes, metadata, created_at
FROM auth_audit_log
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

INSERT INTO password_history (user_id, password_hash)
VALUES ($1, $2);

SELECT password_hash
FROM password_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;
