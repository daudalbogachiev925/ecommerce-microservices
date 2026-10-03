package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Insert — возвращает false, если такое уведомление уже было (идемпотентность)
func (s *Store) Insert(ctx context.Context, userID uuid.UUID, channel, subject, body, sourceEvent, idemKey string) (bool, error) {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO notifications (user_id, channel, subject, body, source_event, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, channel, subject, body, sourceEvent, idemKey,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, nil // уже было
		}
		return false, err
	}
	return true, nil
}
