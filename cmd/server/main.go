package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/config"
	apphttp "github.com/cuffeyvidzro/leamout/internal/http"
	"github.com/cuffeyvidzro/leamout/internal/platform/database"
	"github.com/cuffeyvidzro/leamout/internal/platform/logger"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx := context.Background()

	postgresPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer postgresPool.Close()

	redisClient, err := database.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Error("failed to connect redis", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("failed to close redis client", slog.Any("error", err))
		}
	}()

	serverRegistry := apphttp.NewServer(cfg, log)
	router := serverRegistry.Router()

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server started", slog.String("addr", server.Addr))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	waitForShutdown(log, server)
}

func waitForShutdown(log *slog.Logger, server *http.Server) {
	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-quit

	log.Info("server shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", slog.Any("error", err))
		return
	}

	log.Info("server stopped")
}
