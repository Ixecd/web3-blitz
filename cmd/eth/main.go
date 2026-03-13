package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	btcBlocksMined = prometheus.NewGauge(prometheus.GaugeOpts{Name: "bitcoin_blocks_mined_total", Help: "Total blocks mined"})
	btcBalance     = prometheus.NewGauge(prometheus.GaugeOpts{Name: "bitcoin_balance_btc", Help: "Miner balance"})

	ethBlockNumber = prometheus.NewGauge(prometheus.GaugeOpts{Name: "ethereum_block_number", Help: "Current ETH block height"})
	ethSyncing     = prometheus.NewGauge(prometheus.GaugeOpts{Name: "ethereum_syncing", Help: "Is syncing (1=yes, 0=no)"})
)

func main() {
	prometheus.MustRegister(btcBlocksMined, btcBalance, ethBlockNumber, ethSyncing)

	// BTC
	btcCfg := &rpcclient.ConnConfig{
		Host:         "localhost:18443/wallet/miner",
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	btcClient, err := rpcclient.New(btcCfg, nil)
	if err != nil {
		log.Fatal("BTC 连接失败:", err)
	}
	defer btcClient.Shutdown()

	// ETH
	log.Println("正在连接 Geth (8545)...")
	ethClient, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Fatal("Geth 连接失败! 请确认 Geth 已启动在 8545 端口:", err)
	}
	defer ethClient.Close()
	log.Println("✅ Geth 连接成功!")

	log.Println("🚀 双链矿场已启动 (BTC + Geth)")
	log.Println("📊 Metrics: http://localhost:2112/metrics")

	// Prometheus
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":2112", nil))
	}()

	// 优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sig; cancel() }()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var totalBlocks float64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// BTC
			addr, _ := btcClient.GetNewAddress("")
			_, _ = btcClient.GenerateToAddress(1, addr, nil)
			totalBlocks++
			btcBlocksMined.Set(totalBlocks)

			bal, _ := btcClient.GetBalance("*")
			btcBalance.Set(bal.ToBTC())

			// ETH
			block, err := ethClient.BlockNumber(context.Background())
			if err != nil {
				log.Println("ETH 获取块高失败:", err)
			} else {
				ethBlockNumber.Set(float64(block))
			}

			log.Printf("⛏️ BTC: %.2f | ETH Block: %d", bal.ToBTC(), block)
		}
	}
}
