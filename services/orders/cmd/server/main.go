package main

import (
	"context"
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
		log.Error("nats connect", "err",
