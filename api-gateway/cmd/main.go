package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fabiankaraben/go-core-banking-platform/api-gateway/internal/config"
	"github.com/fabiankaraben/go-core-banking-platform/api-gateway/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.Load()

	rateLimiter := middleware.NewRateLimiter(cfg.RedisAddr, cfg.RateLimitRPM, logger)
	defer rateLimiter.Close()

	accountURL, err := url.Parse(cfg.AccountServiceURL)
	if err != nil {
		logger.Fatal("invalid account service URL", zap.Error(err))
	}
	transferURL, err := url.Parse(cfg.TransferServiceURL)
	if err != nil {
		logger.Fatal("invalid transfer service URL", zap.Error(err))
	}

	accountProxy := httputil.NewSingleHostReverseProxy(accountURL)
	transferProxy := httputil.NewSingleHostReverseProxy(transferURL)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(middleware.CorrelationID)
	r.Use(rateLimiter.Limit)

	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Route("/api/v1", func(r chi.Router) {
		r.Handle("/accounts", accountProxy)
		r.Handle("/accounts/*", accountProxy)
		r.Handle("/transfers", transferProxy)
		r.Handle("/transfers/*", transferProxy)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("api-gateway starting", zap.String("port", cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down api-gateway...")
}
