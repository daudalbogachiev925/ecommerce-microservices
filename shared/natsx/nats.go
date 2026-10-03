package natsx

import (
	"context"
	"time"

	"github.com/daudalobogachiev925/ecommerce-microservices/shared/events"
	"github.com/nats-io/nats.go"
)

type Client struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

func Connect(url string) (*Client, error) {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.Name("ecom-service"),
	)
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	return &Client{conn: nc, js: js}, nil
}

func (c *Client) Close() { c.conn.Close() }

// EnsureStream создаёт стрим с ретеншеном, чтобы события не терялись при рестарте
func (c *Client) EnsureStream(name string, subjects []string) error {
	_, err := c.js.AddStream(&nats.StreamConfig{
		Name:      name,
		Subjects:  subjects,
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
		Storage:   nats.FileStorage,
	})
	return err
}

func (c *Client) Publish(ctx context.Context, subject string, env *events.Envelope) error {
	data, err := env.Encode()
	if err != nil {
		return err
	}
	_, err = c.js.Publish(subject, data, nats.Context(ctx))
	return err
}

// Subscribe — durable consumer, at-least-once. Обработчик должен быть идемпотентным.
func (c *Client) Subscribe(subject, durable string, handler func(*events.Envelope) error) error {
	_, err := c.js.Subscribe(subject, func(m *nats.Msg) {
		env, err := events.Decode(m.Data)
		if err != nil {
			_ = m.Term() // невалидное сообщение — не ретраить, отправить в dead
			return
		}
		if err := handler(env); err != nil {
			_ = m.Nak() // временная ошибка — пусть переотправит
			return
		}
		_ = m.Ack()
	}, nats.Durable(durable), nats.ManualAck(), nats.AckWait(30*time.Second))
	return err
}
