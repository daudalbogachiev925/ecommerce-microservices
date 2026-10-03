package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Payment struct {
	ID          uuid.UUID `json:"id"`
	OrderID     uuid.UUID `json:"order_id"`
	UserID      uuid.UUID `json:"user_id"`
	AmountCents int64     `json:"amount_cents"`
	Status      string    `json:"status"`
	Reason      string    `json:"reason,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

const (
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Insert — возвращает false, если платёж по этому заказу уже есть (идемпотентность)
func (s *Store) Insert(ctx context.Context, orderID, userID uuid.UUID, amount int64, status, reason string) (*Payment, bool, error) {
	var p Payment
	err := s.pool.QueryRow(ctx,
		`INSERT INTO payments (order_id, user_id, amount_cents, status, reason)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, order_id, user_id, amount_cents, status, COALESCE(reason,''), created_at`,
		orderID, userID, amount, status, reason,
	).Scan(&p.ID, &p.OrderID, &p.UserID, &p.AmountCents, &p.Status, &p.Reason, &p.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &p, true, nil
}

func (s *Store) FindByOrder(ctx context.Context, orderID uuid.UUID) (*Payment, error) {
	var p Payment
	err := s.pool.QueryRow(ctx,
		`SELECT id, order_id, user_id, amount_cents, status, COALESCE(reason,''), created_at
		 FROM payments WHERE order_id = $1`, orderID,
	).Scan(&p.ID, &p.OrderID, &p.UserID, &p.AmountCents, &p.Status, &p.Reason, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func isUniqueViolation(err error) bool {
	// pgx возвращает ошибку в pgconn.PgError. Не тащим зависимость — проверяем по строке.
	// В проде — errors.As(err, &pgErr) && pgErr.Code == "23505"
	return err != nil && contains(err.Error(), "23505")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
