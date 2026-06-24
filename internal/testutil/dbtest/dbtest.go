package dbtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var schemaNameRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func NewPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping Postgres integration test")
	}

	ctx := context.Background()
	schema := testSchema(t.Name())

	adminCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminCfg)
	if err != nil {
		t.Fatalf("connect test postgres: %v", err)
	}
	defer adminPool.Close()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, schema)); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = adminPool.Exec(ctx, fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, schema))
	})

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	cfg.MaxConns = 4
	cfg.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect isolated test postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	ApplyMigrations(t, pool)
	return pool
}

func ApplyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	var files []string
	var err error
	for _, pattern := range []string{"migrations/*.sql", "../../migrations/*.sql", "../../../migrations/*.sql"} {
		files, err = filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob migrations: %v", err)
		}
		if len(files) > 0 {
			break
		}
	}
	if len(files) == 0 {
		t.Fatal("no migrations found")
	}
	sort.Strings(files)

	ctx := context.Background()
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := pool.Exec(ctx, upMigrationSQL(string(data))); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(file), err)
		}
	}
}

func upMigrationSQL(contents string) string {
	parts := strings.Split(contents, "-- +goose Down")
	return strings.ReplaceAll(parts[0], "-- +goose Up", "")
}

func testSchema(name string) string {
	sanitized := strings.ToLower(schemaNameRE.ReplaceAllString(name, "_"))
	sanitized = strings.Trim(sanitized, "_")
	if sanitized == "" {
		sanitized = "test"
	}
	if len(sanitized) > 40 {
		sanitized = sanitized[:40]
	}
	return fmt.Sprintf("test_%s_%d", sanitized, time.Now().UnixNano())
}
