package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable"
	}

	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := runMigrations(database); err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	slog.Info("数据库已连接")
	return database, nil
}

func runMigrations(database *sql.DB) error {
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "internal/db/migrations"
	}

	driver, err := postgres.WithInstance(database, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	v, _, _ := m.Version()
	slog.Info("数据库迁移完成", "version", v)
	return nil
}

func runSeed(db *sql.DB) error {
	seed := `
INSERT INTO withdrawal_limits (level, level_name, btc_daily, eth_daily, min_deposit) VALUES
(0, '普通用户',  '2.00000000',   '50.00000000',   '0.00000000'),
(1, '白银用户',  '10.00000000',  '200.00000000',  '1.00000000'),
(2, '黄金用户',  '50.00000000',  '1000.00000000', '10.00000000'),
(3, '钻石用户',  '200.00000000', '5000.00000000', '50.00000000')
ON CONFLICT (level) DO NOTHING;`
	_, err := db.Exec(seed)
	return err
}
