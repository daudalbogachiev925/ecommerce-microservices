package processor

import (
	"context"
	"log/slog"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/payments/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/events"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/natsx"
	"github.com/google/uuid"
)

type Processor struct {
	Store  *store.Store
	Bus    *natsx.Client
	Logger *slog.Logger
}

func New(s *store.Store, b *natsx.Client, l *slog.Logger) *Processor {
	return &Processor{Store: s, Bus: b, Logger: l}
}

// HandleOrderCreated — вызывается из подписчика order.created.
//
// Stub-логика "провайдера": если amount_cents заканчивается на 99 — платёж отклоняется.
// Это позволяет тестировать Saga fail-ветку воспроизводимо.
//
// ВОПРОС: почему детерминированно, а не random?
// Ответ: потому что random делает тесты нестабильными. Детерминированный триггер
// (last two digits == 99) воспроизводим и понятен: 12999 упадёт, 12900 пройдёт.
func (p *Processor) HandleOrderCreated(ctx context.Context, ev events.OrderCreated) error {
	orderID, err := uuid.Parse(ev.OrderID)
	if err != nil {
		return nil
	}
	userID, err := uuid.Parse(ev.UserID)
	if err != nil {
		return nil
	}

	// идемпотентность: если платёж по заказу уже есть — не обрабатываем повторно
	existing, err := p.Store.FindByOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if existing != nil {
		p.Logger.Info("payment already exists, skip", "order_id", orderID)
		return nil
	}

	status := store.StatusSucceeded
	reason := ""
	if ev.TotalCents%100 == 99 {
		status = store.StatusFailed
		reason = "card_declined: issuer refused (test trigger)"
	}

	_, inserted, err := p.Store.Insert(ctx, orderID, userID, ev.TotalCents, status, reason)
	if err != nil {
		return err
	}
	if !inserted {
		return nil // гонка — кто-то другой уже записал
	}

	if status == store.StatusSucceeded {
		env, _ := events.NewEnvelope(events.SubjectPaymentSucceeded, events.PaymentSucceeded{
			OrderID: ev.OrderID,
		})
		return p.Bus.Publish(ctx, events.SubjectPaymentSucceeded, env)
	}

	env, _ := events.NewEnvelope(events.SubjectPaymentFailed, events.PaymentFailed{
		OrderID: ev.OrderID,
		Reason:  reason,
	})
	return p.Bus.Publish(ctx, events.SubjectPaymentFailed, env)
}
