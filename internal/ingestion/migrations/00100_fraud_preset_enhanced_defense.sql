-- +goose Up
-- +goose StatementBegin
UPDATE fraud_policy_presets SET name = 'enhanced_defense' WHERE name = 'gray_market';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE fraud_policy_presets SET name = 'gray_market' WHERE name = 'enhanced_defense';
-- +goose StatementEnd
