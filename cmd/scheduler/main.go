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
	platformcron "github.com/cuffeyvidzro/leamout/internal/platform/cron"
	"github.com/cuffeyvidzro/leamout/internal/platform/database"
	"github.com/cuffeyvidzro/leamout/internal/platform/logger"
	"github.com/cuffeyvidzro/leamout/internal/platform/queue"
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

	riverClient, err := queue.NewClient(postgresPool, nil, queue.Config{Enabled: false})
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

	dunningRepository := dunning.NewRepository(postgresPool)
	scanner := dunning.NewScanner(dunningRepository, riverClient.River, log)

	_, err = scheduler.AddJob(platformcron.ScheduleHourly, func() {
		if inserted, err := scanner.RunOnce(context.Background(), time.Now().UTC()); err != nil {
			log.Error("renewal scanner failed", slog.Any("error", err))
		} else {
			log.Info("renewal scanner enqueued jobs", slog.Int("jobs_inserted", inserted))
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
