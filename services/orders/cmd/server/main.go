package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/orders/internal/clients"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/orders/internal/handlers"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/orders/internal/saga"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/orders/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/dbx"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/events"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/logger"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/natsx"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log := logger.New("orders")

	db, err := dbx.Connect(ctx, env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/orders?sslmode=disable"))
	if err != nil {
		log.Error("db connect", "err", err)
		os.Exit(1)
	}

	bus, err := natsx.Connect(env("NATS_URL", "nats://localhost:4222"))
	if err != nil {
		log.Error("nats connect", "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	_ = bus.EnsureStream("ORDER_EVENTS", []string{"payment.>", "order.>"})

	st := store.New(db)
	catClient := clients.NewCatalog(env("CATALOG_URL", "http://localhost:8082"))
	cartClient := clients.NewCart(env("CART_URL", "http://localhost:8083"))
	orch := saga.New(st, catClient, cartClient, bus, log)
	h := handlers.New(st, orch, log)

	_ = bus.Subscribe(events.SubjectPaymentSucceeded, "orders-payment-ok", func(env *events.Envelope) error {
		var ev events.PaymentSucceeded
		if err := json.Unmarshal(env.Payload, &ev); err != nil {
			log.Error("decode payment.succeeded", "err", err)
			return nil
		}
		return orch.HandlePaymentSucceeded(ctx, ev)
	})

	_ = bus.Subscribe(events.SubjectPaymentFailed, "orders-payment-fail", func(env *events.Envelope) error {
		var ev events.PaymentFailed
		if err := json.Unmarshal(env.Payload, &ev); err != nil {
			log.Error("decode payment.failed", "err", err)
			return nil
		}
		return orch.HandlePaymentFailed(ctx, ev)
	})

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Post("/orders", h.Create)
	r.Get("/orders", h.List)
	r.Get("/orders/{id}", h.Get)

	srv := &http.Server{
		Addr:              env("HTTP_ADDR", ":8084"),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("serve", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	_ = srv.Shutdown(shutCtx)
	log.Info("shutdown")
}
