-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS social_in_app_enabled BOOLEAN NOT NULL DEFAULT false;

INSERT INTO fraud_policy_presets (name, pass, suspect, ivt, block) VALUES
    ('social_in_app', 30, 60, 80, 100)
ON CONFLICT (name) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM fraud_policy_presets WHERE name = 'social_in_app';

ALTER TABLE campaigns
    DROP COLUMN IF EXISTS social_in_app_enabled;
-- +goose StatementEnd
