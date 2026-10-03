CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS products (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku         TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    stock       INT NOT NULL DEFAULT 0 CHECK (stock >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- GIN-индекс для быстрого ILIKE по названию
CREATE INDEX IF NOT EXISTS idx_products_name_trgm
    ON products USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_products_sku ON products (sku);

-- сидим немного товаров, чтобы корзина и заказы сразу работали
INSERT INTO products (sku, name, description, price_cents, stock) VALUES
  ('SKU-001', 'Mechanical Keyboard', 'Hot-swappable, RGB', 12900, 50),
  ('SKU-002', 'Wireless Mouse', 'Ergonomic, 2.4GHz', 4500, 100),
  ('SKU-003', '27" 4K Monitor', 'IPS, 60Hz, HDR', 39900, 20),
  ('SKU-004', 'USB-C Hub', '7-in-1, HDMI, SD', 5900, 75),
  ('SKU-005', 'Noise-cancelling Headphones', 'Over-ear, 30h', 24900, 30),
  ('SKU-006', 'Laptop Stand', 'Aluminium, adjustable', 3900, 60),
  ('SKU-007', 'Webcam 1080p', 'Autofocus, built-in mic', 6900, 40),
  ('SKU-008', 'External SSD 1TB', 'USB 3.2, 1050MB/s', 11900, 25)
ON CONFLICT (sku) DO NOTHING;
