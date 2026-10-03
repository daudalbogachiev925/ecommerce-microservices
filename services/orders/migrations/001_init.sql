CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS orders (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    status      TEXT NOT NULL,          -- pending | paid | confirmed | cancelled
    total_cents BIGINT NOT NULL,
    reason      TEXT,                   -- причина отмены
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_user ON orders (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);

CREATE TABLE IF NOT EXISTS order_items (
    order_id     UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id   UUID NOT NULL,
    quantity     INT NOT NULL,
    price_cents  BIGINT NOT NULL,
    PRIMARY KEY (order_id, product_id)
);

-- saga_log: для отладки и для "что если процесс упал между шагами"
CREATE TABLE IF NOT EXISTS saga_log (
    id          BIGSERIAL PRIMARY KEY,
    order_id    UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    step        TEXT NOT NULL,   -- reserve_stock | request_payment | confirm | release_stock | cancel
    status      TEXT NOT NULL,   -- ok | failed
    message     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_saga_log_order ON saga_log (order_id, created_at);
