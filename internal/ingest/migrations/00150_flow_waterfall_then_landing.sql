-- +goose Up
-- +goose StatementBegin
ALTER TABLE flows
    DROP CONSTRAINT IF EXISTS flows_flow_routing_mode_check;

ALTER TABLE flows
    ADD CONSTRAINT flows_flow_routing_mode_check
    CHECK (flow_routing_mode IN ('weighted', 'waterfall', 'waterfall_then_landing'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE flows
    DROP CONSTRAINT IF EXISTS flows_flow_routing_mode_check;

ALTER TABLE flows
    ADD CONSTRAINT flows_flow_routing_mode_check
    CHECK (flow_routing_mode IN ('weighted', 'waterfall'));
-- +goose StatementEnd
