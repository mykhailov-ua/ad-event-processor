-- +goose Up
-- +goose StatementBegin
ALTER TYPE campaign_status_type ADD VALUE IF NOT EXISTS 'ARCHIVED';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
