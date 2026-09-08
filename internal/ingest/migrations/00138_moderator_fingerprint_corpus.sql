-- Moderator fingerprint corpus tuples for review-traffic safe-page routing.
CREATE TABLE IF NOT EXISTS fraud_moderator_corpus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ja3 TEXT NOT NULL,
    ja4 TEXT NOT NULL DEFAULT '',
    tcp_sig TEXT NOT NULL DEFAULT '',
    webgl_renderer TEXT NOT NULL DEFAULT '',
    layer_desync_count SMALLINT NOT NULL DEFAULT 0,
    note TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fraud_moderator_corpus_ja3_nonempty CHECK (length(trim(ja3)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS fraud_moderator_corpus_tuple_uidx
    ON fraud_moderator_corpus (ja3, ja4, tcp_sig, webgl_renderer, layer_desync_count);
