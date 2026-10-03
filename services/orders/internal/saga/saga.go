package saga

import (
	"context"
	"log/slog"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/orders/internal/clients"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/orders/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/events"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/natsx"
)

// Orchestrator реализует Saga для создания заказа.
//
// Шаги:
//   1. reserve stock в каталоге (по каждому товару)
//   2. publicar order.created → payments подхватит
//   3. ждём payment.succeeded или payment.failed (события)
//   4a. success: confirm order, clear cart, notify
//   4b. fail:    release stock (компенсация), cancel order
//
// ВОПРОС на интервью: почему order.created через событие, а reserve stock через HTTP?
// Ответ: reserve критичен и нужен немедленный ответ (fail fast). payment — асинхронный,
// там ок пауза в 1-2 секунды. Такая гибридная схема — сознательный трейд-офф.
type Orchestrator struct {
	Store   *store.Store
	Catalog *clients.Catalog
	Cart    *clients.Cart
	Bus     *natsx.Client
	Logger  *slog.Logger
}

func New(s *store.Store, c *clients.Catalog, ct *clients.Cart, b *natsx.Client, l *slog.Logger) *Orchestrator {
	return &Orchestrator{Store: s, Catalog: c, Cart: ct, Bus: b, Logger: l}
}

// StartOrder запускает saga и возвращает заказ в статусе pending.
// Ошибки reserve stock возвращаются сразу — заказ не создаётся.
func (o *Orchestrator) StartOrder(ctx context.Context, userID string, items []store.OrderItem) (*store.Order, error) {
	uid := mustUUID(userID)

	var total int64
	for _, it := range items {
		total += it.PriceCents * int64(it.Quantity)
	}

	order, err := o.Store.Create(ctx, uid, items, total)
	if err != nil {
		return nil, err
	}
	o.Store.LogStep(ctx, order.ID, "create_order", "ok", "")

	// Шаг 1: reserve stock. Если хоть один падает — компенсируем уже зарезервированное.
	if err := o.reserveAll(ctx, order.ID, items); err != nil {
		o.Store.LogStep(ctx, order.ID, "reserve_stock", "failed", err.Error())
		o.Store.UpdateStatus(ctx, order.ID, store.StatusCancelled, "stock unavailable: "+err.Error())
		return order, err
	}
	o.Store.LogStep(ctx, order.ID, "reserve_stock", "ok", "")

	// Шаг 2: публикуем order.created — payments и notifications подхватят
	env, _ := events.NewEnvelope(events.SubjectOrderCreated, events.OrderCreated{
		OrderID:    order.ID.String(),
		UserID:     uid.String(),
		Items:      toEventItems(items),
		TotalCents: total,
	})
	if err := o.Bus.Publish(ctx, events.SubjectOrderCreated, env); err != nil {
		// компенсируем reserve и отменяем
		o.Store.LogStep(ctx, order.ID, "publish_order_created", "failed", err.Error())
		o.releaseAll(ctx, order.ID, items)
		o.Store.UpdateStatus(ctx, order.ID, store.StatusCancelled, "publish failed")
		return order, err
	}
	o.Store.LogStep(ctx, order.ID, "publish_order_created", "ok", "")

	return order, nil
}

// HandlePaymentSucceeded — вызывается из подписчика NATS
func (o *Orchestrator) HandlePaymentSucceeded(ctx context.Context, ev events.PaymentSucceeded) error {
	orderID := mustUUID(ev.OrderID)
	order, err := o.Store.Get(ctx, orderID)
	if err != nil || order == nil {
		return nil // нечего делать
	}
	// идемпотентность: если уже confirmed — не повторяем
	if order.Status == store.StatusConfirmed {
		return nil
	}

	if err := o.Store.UpdateStatus(ctx, orderID, store.StatusConfirmed, ""); err != nil {
		return err
	}
	o.Store.LogStep(ctx, orderID, "confirm", "ok", "")

	// очищаем корзину
	if err := o.Cart.Clear(ctx, order.UserID); err != nil {
		// не критично: заказ подтверждён, корзина очистится вручную или потом
		o.Store.LogStep(ctx, orderID, "clear_cart", "failed", err.Error())
	} else {
		o.Store.LogStep(ctx, orderID, "clear_cart", "ok", "")
	}

	// публикуем order.confirmed для notifications
	env, _ := events.NewEnvelope(events.SubjectOrderConfirmed, events.OrderConfirmed{
		OrderID: orderID.String(),
		UserID:  order.UserID.String(),
	})
	_ = o.Bus.Publish(ctx, events.SubjectOrderConfirmed, env)

	return nil
}

// HandlePaymentFailed — компенсирующая ветка Saga
func (o *Orchestrator) HandlePaymentFailed(ctx context.Context, ev events.PaymentFailed) error {
	orderID := mustUUID(ev.OrderID)
	order, err := o.Store.Get(ctx, orderID)
	if err != nil || order == nil {
		return nil
	}
	if order.Status == store.StatusCancelled {
		return nil // уже отменён
	}

	// компенсация: вернуть сток
	o.releaseAll(ctx, orderID, order.Items)
	o.Store.LogStep(ctx, orderID, "release_stock", "ok", "compensated")

	if err := o.Store.UpdateStatus(ctx, orderID, store.StatusCancelled, "payment failed: "+ev.Reason); err != nil {
		return err
	}
	o.Store.LogStep(ctx, orderID, "cancel", "ok", ev.Reason)

	env, _ := events.NewEnvelope(events.SubjectOrderCancelled, events.OrderCancelled{
		OrderID: orderID.String(),
		Reason:  ev.Reason,
	})
	_ = o.Bus.Publish(ctx, events.SubjectOrderCancelled, env)
	return nil
}

// reserveAll — резервирует по каждому товару. При падении откатывает уже зарезервированное.
func (o *Orchestrator) reserveAll(ctx context.Context, orderID interface{}, items []store.OrderItem) error {
	var reserved []store.OrderItem
	for _, it := range items {
		if err := o.Catalog.Reserve(ctx, it.ProductID, it.Quantity); err != nil {
			// компенсация: вернуть то, что успели зарезервировать
			for _, r := range reserved {
				_ = o.Catalog.Release(ctx, r.ProductID, r.Quantity)
			}
			return err
		}
		reserved = append(reserved, it)
	}
	return nil
}

func (o *Orchestrator) releaseAll(ctx context.Context, orderID interface{}, items []store.OrderItem) {
	for _, it := range items {
		// игнорируем ошибки — компенсация должна быть best-effort,
		// иначе одна ошибка сломает весь rollback
		_ = o.Catalog.Release(ctx, it.ProductID, it.Quantity)
	}
}

func toEventItems(items []store.OrderItem) []events.OrderItem {
	out := make([]events.OrderItem, 0, len(items))
	for _, it := range items {
		out = append(out, events.OrderItem{
			ProductID:  it.ProductID.String(),
			Quantity:   it.Quantity,
			PriceCents: it.PriceCents,
		})
	}
	return out
}

func mustUUID(s string) (u uuidLike) {
	id, err := parseUUID(s)
	if err != nil {
		return u
	}
	return u(id)
}
