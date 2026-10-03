package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Catalog struct {
	baseURL string
	http    *http.Client
}

func NewCatalog(baseURL string) *Catalog {
	return &Catalog{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Catalog) Reserve(ctx context.Context, productID uuid.UUID, qty int) error {
	return c.post(ctx,
		fmt.Sprintf("%s/internal/products/%s/reserve", c.baseURL, productID),
		map[string]int{"quantity": qty})
}

func (c *Catalog) Release(ctx context.Context, productID uuid.UUID, qty int) error {
	return c.post(ctx,
		fmt.Sprintf("%s/internal/products/%s/release", c.baseURL, productID),
		map[string]int{"quantity": qty})
}

func (c *Catalog) post(ctx context.Context, url string, body any) error {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("catalog: status %d", resp.StatusCode)
	}
	return nil
}

type Cart struct {
	baseURL string
	http    *http.Client
}

func NewCart(baseURL string) *Cart {
	return &Cart{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Clear очищает корзину — вызывается после подтверждения заказа
func (c *Cart) Clear(ctx context.Context, userID uuid.UUID) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/cart", c.baseURL), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-User-ID", userID.String())

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("cart: status %d", resp.StatusCode)
	}
	return nil
}
