-- +goose Up
-- +goose StatementBegin

ALTER TABLE events ADD COLUMN IF NOT EXISTS ip_hash BYTEA;

ALTER TABLE events DROP CONSTRAINT IF EXISTS events_ip_hash_len;
ALTER TABLE events ADD CONSTRAINT events_ip_hash_len
    CHECK (ip_hash IS NULL OR octet_length(ip_hash) = 16);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE events DROP CONSTRAINT IF EXISTS events_ip_hash_len;
ALTER TABLE events DROP COLUMN IF EXISTS ip_hash;

-- +goose StatementEnd
