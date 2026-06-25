package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cuffeyvidzro/leamout/internal/config"
	"github.com/cuffeyvidzro/leamout/internal/modules/dunning"
	"github.com/cuffeyvidzro/leamout/internal/platform/database"
	"github.com/cuffeyvidzro/leamout/internal/platform/logger"
	"github.com/cuffeyvidzro/leamout/internal/platform/queue"
	"github.com/riverqueue/river"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	if !cfg.Queue.Enabled {
		log.Info("queue worker is disabled")
		return
	}

	ctx := context.Background()

	postgresPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer postgresPool.Close()

	dunningRepository := dunning.NewRepository(postgresPool)
	dunningService := dunning.NewService(dunningRepository, nil)
	workers := river.NewWorkers()
	river.AddWorker(workers, dunning.NewSendRenewalSMSWorker(dunningService, dunningRepository, cfg.ShortBaseURL, log))

	queueClient, err := queue.NewClient(postgresPool, workers, queue.Config{
		Enabled:    cfg.Queue.Enabled,
		MaxWorkers: cfg.Queue.MaxWorkers,
	})
	if err != nil {
		log.Error("failed to create queue client", slog.Any("error", err))
		os.Exit(1)
	}

	if err := queueClient.Start(ctx); err != nil {
		log.Error("failed to start queue worker", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info(
		"queue worker started",
		slog.Int("max_workers", cfg.Queue.MaxWorkers),
	)

	waitForShutdown(log, queueClient)
}

func waitForShutdown(log *slog.Logger, queueClient *queue.Client) {
	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-quit

	log.Info("queue worker shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := queueClient.Stop(ctx); err != nil {
		log.Error("failed to stop queue worker", slog.Any("error", err))
		return
	}

	log.Info("queue worker stopped")
}
