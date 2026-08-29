-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns RENAME COLUMN ghost_ivt_enabled TO silent_reject_enabled;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns RENAME COLUMN silent_reject_enabled TO ghost_ivt_enabled;
-- +goose StatementEnd
