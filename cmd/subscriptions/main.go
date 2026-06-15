package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"effective-mobile-subscriptions/internal/config"
	"effective-mobile-subscriptions/internal/httpapi"
	"effective-mobile-subscriptions/internal/migrations"
	"effective-mobile-subscriptions/internal/repository"
	"effective-mobile-subscriptions/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := migrations.Run(ctx, pool, logger); err != nil {
		logger.Error("run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	repo := repository.NewPostgres(pool)
	svc := service.New(repo)
	handler := httpapi.NewRouter(svc, logger, cfg.APIKey)

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("http server started", slog.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", slog.String("error", err.Error()))
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown http server", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("http server stopped")
}
