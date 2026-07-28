-- +goose Up
-- +goose StatementBegin
CREATE TABLE experiment_cohorts (
  id         UUID PRIMARY KEY,
  name       TEXT NOT NULL,
  active     BOOLEAN NOT NULL DEFAULT TRUE,
  salt       TEXT NOT NULL,
  variants   JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX experiment_cohorts_name_uidx ON experiment_cohorts (LOWER(name));
CREATE INDEX experiment_cohorts_active_idx ON experiment_cohorts (active) WHERE active = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS experiment_cohorts;
-- +goose StatementEnd
