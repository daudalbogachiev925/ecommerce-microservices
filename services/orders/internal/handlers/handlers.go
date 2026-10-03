package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/orders/internal/saga"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/orders/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/httpjson"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handlers struct {
	Store *store.Store
	Saga  *saga.Orchestrator
	Log   *slog.Logger
}

func New(s *store.Store, sg *saga.Orchestrator, l *slog.Logger) *Handlers {
	return &Handlers{Store: s, Saga: sg, Log: l}
}

type createReq struct {
	Items []struct {
		ProductID  string `json:"product_id"`
		Quantity   int    `json:"quantity"`
		PriceCents int64  `json:"price_cents"`
	} `json:"items"`
}

// Create — POST /orders
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	uid, err := uuid.Parse(r.Header.Get("X-User-ID"))
	if err != nil {
		httpjson.Error(w, 401, "unauthorized")
		return
	}

	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Items) == 0 {
		httpjson.Error(w, 400, "items required")
		return
	}

	items := make([]store.OrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		pid, err := uuid.Parse(it.ProductID)
		if err != nil {
			httpjson.Error(w, 400, "invalid product_id")
			return
		}
		if it.Quantity <= 0 || it.PriceCents < 0 {
			httpjson.Error(w, 400, "invalid quantity or price")
			return
		}
		items = append(items, store.OrderItem{
			ProductID:  pid,
			Quantity:   it.Quantity,
			PriceCents: it.PriceCents,
		})
	}

	order, err := h.Saga.StartOrder(r.Context(), uid, items)
	if err != nil {
		// заказ создан, но reserve/publish упал — статус уже cancelled
		if order != nil {
			httpjson.Write(w, 409, order)
			return
		}
		httpjson.Error(w, 500, "order failed")
		return
	}
	httpjson.Write(w, 201, order)
}

// Get — GET /orders/{id}
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	uid, err := uuid.Parse(r.Header.Get("X-User-ID"))
	if err != nil {
		httpjson.Error(w, 401, "unauthorized")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpjson.Error(w, 400, "invalid id")
		return
	}
	o, err := h.Store.Get(r.Context(), id)
	if err != nil || o == nil {
		httpjson.Error(w, 404, "not found")
		return
	}
	// не даём смотреть чужие заказы
	if o.UserID != uid {
		httpjson.Error(w, 403, "forbidden")
		return
	}
	httpjson.Write(w, 200, o)
}

// List — GET /orders
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	uid, err := uuid.Parse(r.Header.Get("X-User-ID"))
	if err != nil {
		httpjson.Error(w, 401, "unauthorized")
		return
	}
	orders, err := h.Store.ListByUser(r.Context(), uid)
	if err != nil {
		httpjson.Error(w, 500, "list failed")
		return
	}
	if orders == nil {
		orders = []store.Order{}
	}
	httpjson.Write(w, 200, orders)
}
