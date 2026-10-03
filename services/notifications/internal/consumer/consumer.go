package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/notifications/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/events"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/natsx"
	"github.com/google/uuid"
)

type Consumer struct {
	Store  *store.Store
	Bus    *natsx.Client
	Logger *slog.Logger
}

func New(s *store.Store, b *natsx.Client, l *slog.Logger) *Consumer {
	return &Consumer{Store: s, Bus: b, Logger: l}
}

// SubscribeAll — вешает подписчиков на все события, которые нам интересны.
//
// ВОПРОС: почему уведомления не вызываются через HTTP из orders?
// Ответ: инверсия зависимостей. orders не должен знать, что у нас есть сервис уведомлений.
// Он публикует факт (order.confirmed) — а кто на это реагирует, его не касается.
// Это позволяет добавлять новых подписчиков (analytics, audit-log, webhooks) без правок в orders.
func (c *Consumer) SubscribeAll(ctx context.Context) error {
	if err := c.Bus.Subscribe(events.SubjectUserRegistered, "notif-user-registered",
		func(env *events.Envelope) error { return c.onUserRegistered(ctx, env) }); err != nil {
		return err
	}
	if err := c.Bus.Subscribe(events.SubjectOrderConfirmed, "notif-order-confirmed",
		func(env *events.Envelope) error { return c.onOrderConfirmed(ctx, env) }); err != nil {
		return err
	}
	if err := c.Bus.Subscribe(events.SubjectOrderCancelled, "notif-order-cancelled",
		func(env *events.Envelope) error { return c.onOrderCancelled(ctx, env) }); err != nil {
		return err
	}
	return nil
}

func (c *Consumer) onUserRegistered(ctx context.Context, env *events.Envelope) error {
	var ev events.UserRegistered
	if err := json.Unmarshal(env.Payload, &ev); err != nil {
		c.Logger.Error("decode user.registered", "err", err)
		return nil
	}
	uid, err := uuid.Parse(ev.UserID)
	if err != nil {
		return nil
	}

	body := fmt.Sprintf("Hi %s! Welcome to our shop. Start browsing the catalog.", ev.Username)
	return c.save(ctx, uid, "email", "Welcome!", body, env)
}

func (c *Consumer) onOrderConfirmed(ctx context.Context, env *events.Envelope) error {
	var ev events.OrderConfirmed
	if err := json.Unmarshal(env.Payload, &ev); err != nil {
		c.Logger.Error("decode order.confirmed", "err", err)
		return nil
	}
	uid, err := uuid.Parse(ev.UserID)
	if err != nil {
		return nil
	}
	body := fmt.Sprintf("Your order %s has been confirmed and is being prepared.", ev.OrderID[:8])
	return c.save(ctx, uid, "email", "Order confirmed", body, env)
}

func (c *Consumer) onOrderCancelled(ctx context.Context, env *events.Envelope) error {
	var ev events.OrderCancelled
	if err := json.Unmarshal(env.Payload, &ev); err != nil {
		c.Logger.Error("decode order.cancelled", "err", err)
		return nil
	}
	// у order.cancelled нет user_id — берём из payload не можем.
	// В реальном проекте — либо добавлять user_id в событие, либо фетчить order по id.
	// Здесь — пропускаем, но фиксируем в лог.
	c.Logger.Info("order cancelled, would notify user",
		"order_id", ev.OrderID, "reason", ev.Reason)
	return nil
}

func (c *Consumer) save(ctx context.Context, userID uuid.UUID, channel, subject, body string, env *events.Envelope) error {
	inserted, err := c.Store.Insert(ctx, userID, channel, subject, body, env.Type, env.IdempotencyKey)
	if err != nil {
		return err
	}
	if !inserted {
		c.Logger.Info("notification already sent, skip", "idem", env.IdempotencyKey)
		return nil
	}
	// здесь был бы реальный вызов SMTP/Push. В демо — лог.
	c.Logger.Info("notification sent",
		"user_id", userID,
		"channel", channel,
		"subject", subject,
	)
	return nil
}
