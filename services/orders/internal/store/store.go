package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Order struct {
	ID         uuid.UUID   `json:"id"`
	UserID     uuid.UUID   `json:"user_id"`
	Status     string      `json:"status"`
	TotalCents int64       `json:"total_cents"`
	Reason     string      `json:"reason,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	Items      []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ProductID  uuid.UUID `json:"product_id"`
	Quantity   int       `json:"quantity"`
	PriceCents int64     `json:"price_cents"`
}

const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusConfirmed = "confirmed"
	StatusCancelled = "cancelled"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Create создаёт заказ и его items атомарно
func (s *Store) Create(ctx context.Context, userID uuid.UUID, items []OrderItem, total int64) (*Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var o Order
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (user_id, status, total_cents)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, status, total_cents, created_at`,
		userID, StatusPending, total,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalCents, &o.CreatedAt)
	if err != nil {
		return nil, err
	}

	for _, it := range items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, price_cents)
			 VALUES ($1, $2, $3, $4)`,
			o.ID, it.ProductID, it.Quantity, it.PriceCents,
		); err != nil {
			return nil, err
		}
	}
	o.Items = items

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (*Order, error) {
	var o Order
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, status, total_cents, COALESCE(reason,''), created_at
		 FROM orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalCents, &o.Reason, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT product_id, quantity, price_cents FROM order_items WHERE order_id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ProductID, &it.Quantity, &it.PriceCents); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, it)
	}
	return &o, rows.Err()
}

func (s *Store) ListByUser(ctx context.Context, userID uuid.UUID) ([]Order, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, status, total_cents, COALESCE(reason,''), created_at
		 FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalCents, &o.Reason, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpdateStatus — переводит заказ в новое состояние
func (s *Store) UpdateStatus(ctx context.Context, id uuid.UUID, status, reason string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE orders SET status = $2, reason = $3, updated_at = now() WHERE id = $1`,
		id, status, reason)
	return err
}

// LogStep — пишем шаг Saga в журнал
func (s *Store) LogStep(ctx context.Context, orderID uuid.UUID, step, status, msg string) {
	_, _ = s.pool.Exec(ctx,
		`INSERT INTO saga_log (order_id, step, status, message)
		 VALUES ($1, $2, $3, $4)`,
		orderID, step, status, msg)
}
