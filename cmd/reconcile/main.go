package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	"github.com/Ixecd/blitz/internal/config"
	"github.com/Ixecd/blitz/internal/db"
	"github.com/Ixecd/blitz/internal/metrics"
	"github.com/Ixecd/blitz/internal/wallet/btc"
	"github.com/Ixecd/blitz/internal/wallet/reconcile"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/push"
)

func main() {
	godotenv.Load()
	metrics.Init()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL environment variable not set")
		os.Exit(1)
	}

	database, err := sql.Open("pgx", dsn)
	if err != nil {
		slog.Error("Failed to open database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		slog.Error("Failed to ping database", "err", err)
		os.Exit(1)
	}

	queries := db.New(database)

	// BTC RPC
	btcRPCHost := os.Getenv("BTC_RPC_HOST")
	if btcRPCHost == "" {
		btcRPCHost = "localhost:18443/wallet/blitz_wallet"
	}
	btcCfg := &rpcclient.ConnConfig{
		Host:         btcRPCHost,
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	btcRPC, err := rpcclient.New(btcCfg, nil)
	if err != nil {
		slog.Error("Failed to connect to BTC RPC", "err", err)
		os.Exit(1)
	}
	btcRPCHolder := config.NewBTCRPCHolder(btcRPC)

	// ETH RPC
	ethRPCHost := os.Getenv("ETH_RPC_HOST")
	if ethRPCHost == "" {
		ethRPCHost = "http://localhost:8545"
	}
	ethRPC, err := ethclient.Dial(ethRPCHost)
	if err != nil {
		slog.Error("Failed to connect to ETH RPC", "err", err)
		os.Exit(1)
	}
	ethRPCHolder := config.NewETHRPCHolder(ethRPC)

	// BTC network
	_ = btc.NetParams()

	reconciler := reconcile.New(queries, btcRPCHolder, ethRPCHolder)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	reconciler.Run(ctx)

	// Push to PushGateway if configured
	if pushURL := os.Getenv("PROMETHEUS_PUSHGATEWAY_URL"); pushURL != "" {
		pusher := push.New(pushURL, "blitz_reconcile").
			Collector(metrics.ReconcileMissingDeposits).
			Collector(metrics.ReconcilePhantomDeposits).
			Collector(metrics.ReconcileAmountMismatch).
			Collector(metrics.ReconcileErrorsTotal).
			Collector(metrics.ReconcileDuration)
		if err := pusher.Push(); err != nil {
			slog.Warn("Failed to push metrics to PushGateway", "err", err)
		}
	}

	slog.Info("Reconciliation run completed")
}
