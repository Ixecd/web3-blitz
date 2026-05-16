package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Ixecd/blitz/internal/config"
	"github.com/Ixecd/blitz/internal/metrics"
	"github.com/Ixecd/blitz/internal/wallet/sweep"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/push"
)

func main() {
	godotenv.Load()
	metrics.Init()

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

	sweeper := sweep.New(btcRPCHolder, ethRPCHolder)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sweeper.Run(ctx)

	// Push to PushGateway if configured
	if pushURL := os.Getenv("PROMETHEUS_PUSHGATEWAY_URL"); pushURL != "" {
		pusher := push.New(pushURL, "blitz_sweep").
			Collector(metrics.SweepBTCTotal).
			Collector(metrics.SweepETHTTotal).
			Collector(metrics.SweepAmountTotal)
		if err := pusher.Push(); err != nil {
			slog.Warn("Failed to push sweep metrics to PushGateway", "err", err)
		}
	}

	slog.Info("Sweep run completed")
}
