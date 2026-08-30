-- +goose Up
CREATE TABLE IF NOT EXISTS command_palette_recents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    item_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    label TEXT NOT NULL,
    href TEXT NOT NULL,
    meta TEXT,
    "group" TEXT,
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (customer_id, user_id, item_id, kind)
);

CREATE INDEX IF NOT EXISTS idx_command_palette_recents_customer_user_accessed
    ON command_palette_recents (customer_id, user_id, accessed_at DESC);

-- +goose Down
DROP TABLE IF EXISTS command_palette_recents;
