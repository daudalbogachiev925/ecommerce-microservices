package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gwmw "github.com/daudalobogachiev925/ecommerce-microservices/services/gateway/internal/middleware"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/gateway/internal/proxy"
	"github.com/daudalobogachiev925/ecommerce-microservices/services/gateway/internal/routes"
	"github.com/daudalobogachiev925/ecommerce-microservices/shared/auth"
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

	log := logger.New("gateway")

	table := routes.Config()
	authSvc := auth.New(env("JWT_SECRET", "dev-secret"))
	reverseProxy := proxy.New(table, log)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(gwmw.Auth(authSvc, table)) // auth первым — до проксирования

	// gateway сам отдаёт health и metrics, не проксирует
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	r.Handle("/metrics", promhttp.Handler())

	// всё остальное — в proxy
	r.Handle("/*", reverseProxy)

	srv := &http.Server{
		Addr:              env("HTTP_ADDR", ":8080"),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("gateway listening", "addr", srv.Addr)
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
