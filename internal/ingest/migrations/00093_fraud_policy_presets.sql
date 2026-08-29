-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS fraud_policy_presets (
    name TEXT PRIMARY KEY,
    pass SMALLINT NOT NULL CHECK (pass BETWEEN 0 AND 100),
    suspect SMALLINT NOT NULL CHECK (suspect BETWEEN 0 AND 100),
    ivt SMALLINT NOT NULL CHECK (ivt BETWEEN 0 AND 100),
    block SMALLINT NOT NULL CHECK (block BETWEEN 0 AND 100),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (pass <= suspect AND suspect <= ivt AND ivt <= block)
);

INSERT INTO fraud_policy_presets (name, pass, suspect, ivt, block) VALUES
    ('conservative', 40, 70, 90, 100),
    ('balanced', 30, 60, 80, 100),
    ('aggressive', 20, 45, 65, 85)
ON CONFLICT (name) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS fraud_policy_presets;
-- +goose StatementEnd
