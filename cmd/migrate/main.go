package main

import (
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL environment variable not set")
		os.Exit(1)
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "/migrations"
	}

	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		slog.Error("Failed to init migrate", "err", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("Migration failed", "err", err)
		os.Exit(1)
	}

	v, dirty, _ := m.Version()
	if dirty {
		slog.Error("Database is in dirty state", "version", v)
		os.Exit(1)
	}

	slog.Info("Migration completed successfully", "version", v)
}
