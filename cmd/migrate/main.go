package main

import (
	"database/sql"
	"log"
	"os"

	leamout "github.com/cuffeyvidzro/leamout"
	"github.com/cuffeyvidzro/leamout/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
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
		log.Fatal("Usage: migrate [up|down|reset|status]")
	}

	command := os.Args[1]

	switch command {
	case "up":
		err = goose.Up(db, "migrations")
	case "down":
		err = goose.Down(db, "migrations")
	case "reset":
		err = goose.Reset(db, "migrations")
	case "status":
		err = goose.Status(db, "migrations")
	default:
		log.Fatalf("unknown command: %s", command)
	}

	if err != nil {
		log.Fatalf("command %s failed: %v", command, err)
	}

	log.Printf("migration command %s completed successfully", command)
}
