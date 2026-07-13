package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) != 2 {
		fatal("usage: migrate up|seed")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://opportunity:opportunity@localhost:5432/opportunity?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		fatal(err.Error())
	}
	defer pool.Close()
	switch os.Args[1] {
	case "up":
		runMigrations(pool)
	case "seed":
		executeFile(pool, "migrations/seed.sql")
	default:
		fatal("unknown command")
	}
}
func runMigrations(pool *pgxpool.Pool) {
	if _, err := pool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		fatal(err.Error())
	}
	files, err := filepath.Glob("migrations/*.up.sql")
	if err != nil {
		fatal(err.Error())
	}
	sort.Strings(files)
	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".up.sql")
		var exists bool
		if err := pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists); err != nil {
			fatal(err.Error())
		}
		if exists {
			continue
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			fatal(err.Error())
		}
		tx, err := pool.Begin(context.Background())
		if err != nil {
			fatal(err.Error())
		}
		if _, err = tx.Exec(context.Background(), string(contents)); err == nil {
			_, err = tx.Exec(context.Background(), "INSERT INTO schema_migrations(version) VALUES($1)", version)
		}
		if err != nil {
			_ = tx.Rollback(context.Background())
			fatal(err.Error())
		}
		if err = tx.Commit(context.Background()); err != nil {
			fatal(err.Error())
		}
		fmt.Println("applied", version)
	}
}
func executeFile(pool *pgxpool.Pool, file string) {
	contents, err := os.ReadFile(file)
	if err != nil {
		fatal(err.Error())
	}
	if _, err := pool.Exec(context.Background(), string(contents)); err != nil {
		fatal(err.Error())
	}
	fmt.Println("executed", file)
}
func fatal(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
