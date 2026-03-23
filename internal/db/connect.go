package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT`)
	db.Exec(`UPDATE users SET email = username WHERE email IS NULL`)
	db.Exec(`ALTER TABLE users ALTER COLUMN email SET NOT NULL`)
	if err := runSeed(db); err != nil {
		return nil, fmt.Errorf("seed db: %w", err)
	}

	var dbName string
	db.QueryRow("SELECT current_database()").Scan(&dbName)
	log.Printf("[DEBUG] DSN: %s", dsn)
	log.Printf("[DEBUG] 实际连接数据库: %s", dbName)

	var hasEmail bool
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='email')").Scan(&hasEmail)
	log.Printf("[DEBUG] users.email 存在: %v", hasEmail)

	return db, nil
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
