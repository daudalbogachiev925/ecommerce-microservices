CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS cart_items (
    user_id      UUID NOT NULL,
    product_id   UUID NOT NULL,
    quantity     INT NOT NULL CHECK (quantity > 0),
    price_cents  BIGINT NOT NULL CHECK (price_cents >= 0),
    added_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_cart_user ON cart_items (user_id);
