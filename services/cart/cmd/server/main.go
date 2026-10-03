package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daudalobogachiev925/ecommerce-microservices/services/cart/internal/handlers"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/cart/internal/store"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/dbx"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/logger"
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

	log := logger.New("cart")

	db, err := dbx.Connect(ctx, env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/cart?sslmode=disable"))
	if err != nil {
		log.Error("db connect", "err", err)
		os.Exit(1)
	}

	st := store.New(db)
	h := handlers.New(st, log)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/cart", h.List)
	r.Post("/cart/items", h.Add)
	r.Patch("/cart/items/{productID}", h.UpdateQty)
	r.Delete("/cart/items/{productID}", h.Remove)
	r.Delete("/cart", h.Clear)

	srv := &http.Server{
		Addr:              env("HTTP_ADDR", ":8083"),
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
