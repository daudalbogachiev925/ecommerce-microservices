package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/cart/internal/store"
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

// userID — gateway уже проверил JWT и проставил X-User-ID
func userID(r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.Header.Get("X-User-ID"))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

type addReq struct {
	ProductID  string `json:"product_id"`
	Quantity   int    `json:"quantity"`
	PriceCents int64  `json:"price_cents"`
}

// Add — POST /cart/items
// ВОПРОС: почему клиент передаёт price_cents, а не сервер берёт из каталога?
// Ответ: price snapshot. Клиент получил цену из GET /products/{id}, и фиксирует её в корзине.
// Сервер доверяет клиенту в этом поле, но перепроверит цену на этапе checkout в orders.
func (h *Handlers) Add(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpjson.Error(w, 401, "unauthorized")
		return
	}

	var req addReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpjson.Error(w, 400, "invalid json")
		return
	}
	pid, err := uuid.Parse(req.ProductID)
	if err != nil {
		httpjson.Error(w, 400, "invalid product_id")
		return
	}
	if req.Quantity <= 0 || req.PriceCents < 0 {
		httpjson.Error(w, 400, "invalid quantity or price")
		return
	}

	if err := h.Store.Add(r.Context(), uid, pid, req.Quantity, req.PriceCents); err != nil {
		h.Logger.Error("add to cart", "err", err)
		httpjson.Error(w, 500, "add failed")
		return
	}
	httpjson.Write(w, 201, map[string]string{"status": "added"})
}

// List — GET /cart
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpjson.Error(w, 401, "unauthorized")
		return
	}

	items, err := h.Store.List(r.Context(), uid)
	if err != nil {
		httpjson.Error(w, 500, "list failed")
		return
	}
	if items == nil {
		items = []store.Item{}
	}
	total, _ := h.Store.Total(r.Context(), uid)
	httpjson.Write(w, 200, map[string]any{
		"items":       items,
		"total_cents": total,
	})
}

// UpdateQty — PATCH /cart/items/{productID}  body: {"quantity": 5}
func (h *Handlers) UpdateQty(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpjson.Error(w, 401, "unauthorized")
		return
	}
	pid, err := uuid.Parse(chi.URLParam(r, "productID"))
	if err != nil {
		httpjson.Error(w, 400, "invalid product_id")
		return
	}
	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpjson.Error(w, 400, "invalid json")
		return
	}

	if err := h.Store.UpdateQuantity(r.Context(), uid, pid, body.Quantity); err != nil {
		httpjson.Error(w, 500, "update failed")
		return
	}
	httpjson.Write(w, 200, map[string]string{"status": "updated"})
}

// Remove — DELETE /cart/items/{productID}
func (h *Handlers) Remove(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpjson.Error(w, 401, "unauthorized")
		return
	}
	pid, err := uuid.Parse(chi.URLParam(r, "productID"))
	if err != nil {
		httpjson.Error(w, 400, "invalid product_id")
		return
	}
	if err := h.Store.Remove(r.Context(), uid, pid); err != nil {
		httpjson.Error(w, 500, "remove failed")
		return
	}
	httpjson.Write(w, 200, map[string]string{"status": "removed"})
}

// Clear — DELETE /cart  (вызывается orders после оформления)
func (h *Handlers) Clear(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpjson.Error(w, 401, "unauthorized")
		return
	}
	if err := h.Store.Clear(r.Context(), uid); err != nil {
		httpjson.Error(w, 500, "clear failed")
		return
	}
	httpjson.Write(w, 200, map[string]string{"status": "cleared"})
}
