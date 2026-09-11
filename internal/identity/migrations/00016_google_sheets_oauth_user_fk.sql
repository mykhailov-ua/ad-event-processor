-- +goose Up
-- +goose StatementBegin
ALTER TABLE google_sheets_oauth_tokens
    ADD CONSTRAINT google_sheets_oauth_tokens_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE google_sheets_oauth_tokens
    DROP CONSTRAINT IF EXISTS google_sheets_oauth_tokens_user_id_fkey;
-- +goose StatementEnd
