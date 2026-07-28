-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_node_capacity_scores_region_role
    ON node_capacity_scores (region_code, role);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_node_capacity_scores_region_role;
-- +goose StatementEnd
