-- +goose Up
-- +goose StatementBegin
ALTER TYPE redis_slot_migration_state ADD VALUE IF NOT EXISTS 'dual_writing' AFTER 'copied';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
