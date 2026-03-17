package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/adapters/kafka"
	nethttp "github.com/fabiankaraben/go-core-banking-platform/account-service/internal/adapters/http"
	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/adapters/postgres"
	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/app"
	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/config"
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

	accountRepo := postgres.NewAccountRepository(db)
	outboxRepo := postgres.NewOutboxRepository(db)

	producer, err := kafka.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		logger.Fatal("creating kafka producer", zap.Error(err))
	}
	defer producer.Close()

	svc := app.NewAccountService(accountRepo, outboxRepo, logger)

	consumer, err := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaGroupID, svc, logger)
	if err != nil {
		logger.Fatal("creating kafka consumer", zap.Error(err))
	}
	defer consumer.Close()

	relay := kafka.NewOutboxRelay(outboxRepo, producer, logger)

	handler := nethttp.NewHandler(svc, logger)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Post("/accounts", handler.CreateAccount)
	r.Get("/accounts/{accountID}", handler.GetAccount)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: r,
	}

	go func() {
		logger.Info("account-service HTTP server starting", zap.String("port", cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server error", zap.Error(err))
		}
	}()

	go consumer.Start(ctx)
	go relay.Start(ctx)

	<-ctx.Done()
	logger.Info("shutting down account-service...")

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
