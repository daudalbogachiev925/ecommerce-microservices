package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/notifications/internal/consumer"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/notifications/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/dbx"
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

	log := logger.New("notifications")

	db, err := dbx.Connect(ctx, env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/notifications?sslmode=disable"))
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

	_ = bus.EnsureStream("USERS", []string{"user.>"})
	_ = bus.EnsureStream("ORDER_EVENTS", []string{"payment.>", "order.>"})

	st := store.New(db)
	cons := consumer.New(st, bus, log)

	if err := cons.SubscribeAll(ctx); err != nil {
		log.Error("subscribe", "err", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              env("HTTP_ADDR", ":8086"),
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
