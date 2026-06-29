package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cuffeyvidzro/leamout/internal/config"
	"github.com/cuffeyvidzro/leamout/internal/modules/subscription"
	platformcron "github.com/cuffeyvidzro/leamout/internal/platform/cron"
	"github.com/cuffeyvidzro/leamout/internal/platform/database"
	"github.com/cuffeyvidzro/leamout/internal/platform/logger"
	"github.com/cuffeyvidzro/leamout/internal/platform/queue"
	dunningworkflow "github.com/cuffeyvidzro/leamout/internal/workflows/dunning"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx := context.Background()

	if !cfg.Cron.Enabled {
		log.Info("cron scheduler is disabled")
		return
	}

	postgresPool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer postgresPool.Close()

	workers := queue.NewWorkerRegistry()
	dunningworkflow.RegisterReminderJobKind(workers)

	riverClient, err := queue.NewClient(postgresPool, workers, queue.Config{Enabled: false})
	if err != nil {
		log.Error("failed to create river client", slog.Any("error", err))
		os.Exit(1)
	}

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

	scheduler, err := platformcron.New(platformcron.Config{
		Enabled:  cfg.Cron.Enabled,
		Timezone: cfg.Cron.Timezone,
	})
	if err != nil {
		log.Error("failed to create cron scheduler", slog.Any("error", err))
		os.Exit(1)
	}

	subscriptionService := subscription.NewService(subscription.NewRepository(postgresPool))
	scanner := dunningworkflow.NewScanner(subscriptionService, func(ctx context.Context, args dunningworkflow.SendReminderArgs) error {
		return riverClient.Insert(ctx, args, nil)
	}, log)

	_, err = scheduler.AddJob(platformcron.ScheduleMin, func() {
		if _, err := scanner.RunOnce(context.Background()); err != nil {
			log.Error("dunning scanner failed", slog.Any("error", err))
		}
	})
	if err != nil {
		log.Error("failed to register cron job", slog.Any("error", err))
		os.Exit(1)
	}

	scheduler.Start()

	log.Info(
		"cron scheduler started",
		slog.String("timezone", cfg.Cron.Timezone),
	)

	waitForShutdown(log, scheduler)
}

func waitForShutdown(log *slog.Logger, scheduler *platformcron.Scheduler) {
	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-quit

	log.Info("cron scheduler shutting down")

	ctx := scheduler.Stop()
	<-ctx.Done()

	log.Info("cron scheduler stopped")
}
