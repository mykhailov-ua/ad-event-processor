-- +goose Up
-- +goose StatementBegin
ALTER TYPE campaign_status_type ADD VALUE IF NOT EXISTS 'DRAINING';
ALTER TYPE campaign_status_type ADD VALUE IF NOT EXISTS 'DELETED';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
