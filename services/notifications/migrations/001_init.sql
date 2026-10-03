CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS notifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    channel       TEXT NOT NULL,        -- email | push | sms
    subject       TEXT NOT NULL,
    body          TEXT NOT NULL,
    source_event  TEXT NOT NULL,        -- user.registered | order.confirmed | ...
    idempotency_key TEXT UNIQUE NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications (user_id, created_at DESC);
