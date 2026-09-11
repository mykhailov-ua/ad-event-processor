-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS google_sheets_oauth_tokens (
    user_id UUID PRIMARY KEY,
    refresh_token_encrypted BYTEA NOT NULL,
    access_token_encrypted BYTEA,
    scopes TEXT NOT NULL DEFAULT '',
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS google_sheets_oauth_tokens;
-- +goose StatementEnd
