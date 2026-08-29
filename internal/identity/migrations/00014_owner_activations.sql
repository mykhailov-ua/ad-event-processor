-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS owner_activations (
    deployment_id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id),
    owner_user_id UUID NOT NULL REFERENCES users(id),
    activated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_owner_activations_customer_id ON owner_activations(customer_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS owner_activations;
-- +goose StatementEnd
