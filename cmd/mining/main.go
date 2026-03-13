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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	blocksMined = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoin_blocks_mined_total",
		Help: "Total blocks mined in regtest",
	})
	balanceGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bitcoin_balance_btc",
		Help: "Current miner wallet balance in BTC",
	})
)

func main() {
	prometheus.MustRegister(blocksMined, balanceGauge)

	cfg := &rpcclient.ConnConfig{
		Host:         "localhost:18443/wallet/miner",
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	client, err := rpcclient.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	log.Println("🚀 私人矿场 + Prometheus 已启动")
	log.Println("📊 Metrics 地址: http://localhost:2112/metrics")
	log.Println("挖矿间隔: 10 秒")

	// 启动 Prometheus exporter
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatal(err)
		}
	}()

	// 优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("⛔ 矿场关闭中...")
		cancel()
	}()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var totalBlocks float64 // 本地变量同步总出块数

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			addr, _ := client.GetNewAddress("")
			_, err := client.GenerateToAddress(1, addr, nil)
			if err != nil {
				log.Println("挖块失败:", err)
				continue
			}

			totalBlocks++
			blocksMined.Set(totalBlocks)

			bal, _ := client.GetBalance("*")
			balanceGauge.Set(bal.ToBTC())

			log.Printf("⛏️ 新块已出 | 余额: %.2f BTC | 总出块: %.0f", bal.ToBTC(), totalBlocks)
		}
	}
}
