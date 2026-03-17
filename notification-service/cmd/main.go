package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fabiankaraben/go-core-banking-platform/notification-service/internal/adapters/kafka"
	"github.com/fabiankaraben/go-core-banking-platform/notification-service/internal/adapters/postgres"
	"github.com/fabiankaraben/go-core-banking-platform/notification-service/internal/app"
	"github.com/fabiankaraben/go-core-banking-platform/notification-service/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, cfg.DBDSN)
	if err != nil {
		logger.Fatal("connecting to database", zap.Error(err))
	}
	defer db.Close()

	if err := runMigrations(cfg.DBDSN); err != nil {
		logger.Fatal("running migrations", zap.Error(err))
	}

	notifRepo := postgres.NewNotificationRepository(db)
	svc := app.NewNotificationService(notifRepo, logger)

	consumer, err := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaGroupID, svc, logger)
	if err != nil {
		logger.Fatal("creating kafka consumer", zap.Error(err))
	}
	defer consumer.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{Addr: fmt.Sprintf(":%s", cfg.HTTPPort), Handler: r}

	go func() {
		logger.Info("notification-service HTTP server starting", zap.String("port", cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server error", zap.Error(err))
		}
	}()

	go consumer.Start(ctx)

	<-ctx.Done()
	logger.Info("shutting down notification-service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
}

func runMigrations(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}
