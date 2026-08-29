-- +goose Up
-- +goose StatementBegin
ALTER TYPE pacing_mode_type ADD VALUE IF NOT EXISTS 'VPP';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
