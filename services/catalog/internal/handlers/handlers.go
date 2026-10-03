package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/catalog/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/httpjson"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handlers struct {
	Store  *store.Store
	Logger *slog.Logger
}

func New(s *store.Store, l *slog.Logger) *Handlers {
	return &Handlers{Store: s, Logger: l}
}

// List — GET /products?q=keyboard&limit=20&offset=0
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	products, err := h.Store.List(r.Context(), q, limit, offset)
	if err != nil {
		h.Logger.Error("list products", "err", err)
		httpjson.Error(w, 500, "list failed")
		return
	}
	if products == nil {
		products = []store.Product{}
	}
	httpjson.Write(w, 200, map[string]any{
		"items": products,
		"count": len(products),
	})
}

// Get — GET /products/{id}
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpjson.Error(w, 400, "invalid id")
		return
	}
	p, err := h.Store.Get(r.Context(), id)
	if err != nil {
		httpjson.Error(w, 500, "lookup failed")
		return
	}
	if p == nil {
		httpjson.Error(w, 404, "not found")
		return
	}
	httpjson.Write(w, 200, p)
}

// ReserveStock — POST /internal/products/{id}/reserve
// вызывается из orders при оформлении заказа
func (h *Handlers) ReserveStock(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpjson.Error(w, 400, "invalid id")
		return
	}

	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := httpjson.Decode(r, &body); err != nil || body.Quantity <= 0 {
		httpjson.Error(w, 400, "quantity required")
		return
	}

	if err := h.Store.ReserveStock(r.Context(), id, body.Quantity); err != nil {
		h.Logger.Warn("reserve stock", "product", id, "err", err)
		httpjson.Error(w, 409, err.Error())
		return
	}
	httpjson.Write(w, 200, map[string]string{"status": "reserved"})
}

// ReleaseStock — POST /internal/products/{id}/release (компенсация в Saga)
func (h *Handlers) ReleaseStock(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpjson.Error(w, 400, "invalid id")
		return
	}
	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := httpjson.Decode(r, &body); err != nil || body.Quantity <= 0 {
		httpjson.Error(w, 400, "quantity required")
		return
	}

	if err := h.Store.ReleaseStock(r.Context(), id, body.Quantity); err != nil {
		httpjson.Error(w, 500, "release failed")
		return
	}
	httpjson.Write(w, 200, map[string]string{"status": "released"})
}
