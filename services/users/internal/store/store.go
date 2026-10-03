package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    string    `json:"created_at"`
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, username, email, hash string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, username, email, password_hash, created_at`,
		username, email, hash,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) FindByUsername(ctx context.Context, username string) (*User, error) {
	return s.findOne(ctx, `SELECT id, username, email, password_hash, created_at
		FROM users WHERE username = $1`, username)
}

func (s *Store) FindByEmail(ctx context.Context, email string) (*User, error) {
	return s.findOne(ctx, `SELECT id, username, email, password_hash, created_at
		FROM users WHERE email = $1`, email)
}

func (s *Store) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.findOne(ctx, `SELECT id, username, email, password_hash, created_at
		FROM users WHERE id = $1`, id)
}

func (s *Store) findOne(ctx context.Context, q string, arg any) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, q, arg).
		Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
