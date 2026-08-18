INSERT INTO segment_members (segment_id, user_hash, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (segment_id, user_hash) DO UPDATE
SET expires_at = EXCLUDED.expires_at,
    added_at = CURRENT_TIMESTAMP;
