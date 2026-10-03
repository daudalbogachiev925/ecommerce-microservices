package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	ProductID  uuid.UUID `json:"product_id"`
	Quantity   int       `json:"quantity"`
	PriceCents int64     `json:"price_cents"`
	AddedAt    time.Time `json:"added_at"`
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Add — идемпотентно: если товар уже в корзине, увеличиваем quantity
func (s *Store) Add(ctx context.Context, userID, productID uuid.UUID, qty int, priceCents int64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO cart_items (user_id, product_id, quantity, price_cents)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, product_id) DO UPDATE
		   SET quantity = cart_items.quantity + EXCLUDED.quantity,
		       updated_at = now()`,
		userID, productID, qty, priceCents)
	return err
}

func (s *Store) UpdateQuantity(ctx context.Context, userID, productID uuid.UUID, qty int) error {
	if qty <= 0 {
		return s.Remove(ctx, userID, productID)
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE cart_items SET quantity = $3, updated_at = now()
		 WHERE user_id = $1 AND product_id = $2`,
		userID, productID, qty)
	return err
}

func (s *Store) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1 AND product_id = $2`,
		userID, productID)
	return err
}

func (s *Store) List(ctx context.Context, userID uuid.UUID) ([]Item, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT product_id, quantity, price_cents, added_at
		 FROM cart_items WHERE user_id = $1 ORDER BY added_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ProductID, &it.Quantity, &it.PriceCents, &it.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Clear — вызывается после успешного оформления заказа
func (s *Store) Clear(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM cart_items WHERE user_id = $1`, userID)
	return err
}

func (s *Store) Total(ctx context.Context, userID uuid.UUID) (int64, error) {
	var total int64
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity * price_cents), 0)
		 FROM cart_items WHERE user_id = $1`, userID).Scan(&total)
	return total, err
}
