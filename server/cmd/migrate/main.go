package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	leamout "github.com/cuffeyvidzro/leamout"
	"github.com/cuffeyvidzro/leamout/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	goose.SetBaseFS(leamout.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to set migration dialect: %v", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate [up|down|reset|status|river-up]")
	}

	command := os.Args[1]

	switch command {
	case "up":
		if err := goose.Up(db, "migrations"); err != nil {
			log.Fatalf("leamout migration up failed: %v", err)
		}

		if err := runRiverMigrations(cfg.DatabaseURL); err != nil {
			log.Fatalf("river migration up failed: %v", err)
		}

	case "down":
		if err := goose.Down(db, "migrations"); err != nil {
			log.Fatalf("leamout migration down failed: %v", err)
		}

	case "reset":
		if err := goose.Reset(db, "migrations"); err != nil {
			log.Fatalf("leamout migration reset failed: %v", err)
		}

		// After resetting app tables, make sure River still has its required tables.
		if err := runRiverMigrations(cfg.DatabaseURL); err != nil {
			log.Fatalf("river migration up failed after reset: %v", err)
		}

	case "status":
		if err := goose.Status(db, "migrations"); err != nil {
			log.Fatalf("leamout migration status failed: %v", err)
		}

	case "river-up":
		if err := runRiverMigrations(cfg.DatabaseURL); err != nil {
			log.Fatalf("river migration up failed: %v", err)
		}

	default:
		log.Fatalf("unknown command: %s", command)
	}

	log.Printf("migration command %s completed successfully", command)
}

func runRiverMigrations(databaseURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}
