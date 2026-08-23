-- +goose Up
-- +goose StatementBegin
INSERT INTO fraud_policy_presets (name, pass, suspect, ivt, block) VALUES
    ('gray_market', 20, 45, 65, 85)
ON CONFLICT (name) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM fraud_policy_presets WHERE name = 'gray_market';
-- +goose StatementEnd
