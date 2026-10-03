package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Subject'ы NATS. Один топик — одно событие. Snake case, точка как разделитель.
const (
	SubjectUserRegistered     = "user.registered"
	SubjectOrderCreated       = "order.created"
	SubjectPaymentRequested   = "payment.requested"
	SubjectPaymentSucceeded   = "payment.succeeded"
	SubjectPaymentFailed      = "payment.failed"
	SubjectOrderConfirmed     = "order.confirmed"
	SubjectOrderCancelled     = "order.cancelled"
	SubjectNotificationSend   = "notification.send"
)

// Envelope — универсальная обёртка. Даёт idempotency_key, чтобы подписчик мог дедуплицировать.
type Envelope struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	IdempotencyKey string          `json:"idempotency_key"`
	OccurredAt     time.Time       `json:"occurred_at"`
	Payload        json.RawMessage `json:"payload"`
}

func NewEnvelope(eventType string, payload any) (*Envelope, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		ID:             uuid.NewString(),
		Type:           eventType,
		IdempotencyKey: uuid.NewString(),
		OccurredAt:     time.Now().UTC(),
		Payload:        b,
	}, nil
}

func (e *Envelope) Encode() ([]byte, error) {
	return json.Marshal(e)
}

func Decode(data []byte) (*Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// Payload'ы конкретных событий — типизированные, чтобы не гонять map[string]any

type UserRegistered struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	PriceCents int64 `json:"price_cents"`
}

type OrderCreated struct {
	OrderID  string      `json:"order_id"`
	UserID   string      `json:"user_id"`
	Items    []OrderItem `json:"items"`
	TotalCents int64     `json:"total_cents"`
}

type PaymentRequested struct {
	OrderID    string `json:"order_id"`
	UserID     string `json:"user_id"`
	AmountCents int64 `json:"amount_cents"`
}

type PaymentSucceeded struct {
	OrderID string `json:"order_id"`
	PaymentID string `json:"payment_id"`
}

type PaymentFailed struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type OrderConfirmed struct {
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

type OrderCancelled struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type NotificationSend struct {
	UserID  string `json:"user_id"`
	Channel string `json:"channel"` // email | push | sms
	Subject string `json:"subject"`
	Body    string `json:"body"`
}
