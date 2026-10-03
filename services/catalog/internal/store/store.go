package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Product struct {
	ID          uuid.UUID `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context, query string, limit, offset int) ([]Product, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	q := `SELECT id, sku, name, description, price_cents, stock, created_at
	      FROM products`
	args := []any{}

	if strings.TrimSpace(query) != "" {
		q += ` WHERE name ILIKE $1 OR description ILIKE $1`
		args = append(args, "%"+query+"%")
	}

	q += ` ORDER BY name LIMIT ` + itoa(limit) + ` OFFSET ` + itoa(offset)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProducts(rows)
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (*Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx,
		`SELECT id, sku, name, description, price_cents, stock, created_at
		 FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCents, &p.Stock, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetMany — bulk-lookup для orders: не хочется делать N запросов при оформлении
func (s *Store) GetMany(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Product, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, sku, name, description, price_cents, stock, created_at
		 FROM products WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[uuid.UUID]Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCents, &p.Stock, &p.CreatedAt); err != nil {
			return nil, err
		}
		out[p.ID] = p
	}
	return out, rows.Err()
}

// ReserveStock — вызывается из orders (через internal API), списывает сток атомарно
func (s *Store) ReserveStock(ctx context.Context, productID uuid.UUID, qty int) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE products SET stock = stock - $2, updated_at = now()
		 WHERE id = $1 AND stock >= $2`,
		productID, qty)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("insufficient stock")
	}
	return nil
}

func (s *Store) ReleaseStock(ctx context.Context, productID uuid.UUID, qty int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE products SET stock = stock + $2, updated_at = now() WHERE id = $1`,
		productID, qty)
	return err
}

func scanProducts(rows pgx.Rows) ([]Product, error) {
	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceCents, &p.Stock, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = digits[i%10]
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
