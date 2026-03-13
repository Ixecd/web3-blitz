package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"
	"os/signal"
	"syscall"

	"github.com/Ixecd/web3-blitz/internal/wallet/btc"
	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/eth"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	walletAddressesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "wallet_addresses_generated_total",
		Help: "Total addresses generated",
	})
)

func main() {
	prometheus.MustRegister(walletAddressesTotal)

	// 初始化 HD Wallet
	hdWallet, err := core.NewHDWallet([]byte("test-seed-for-dev-only-1234567890"))
	if err != nil {
		log.Fatal("HDWallet 初始化失败:", err)
	}

	// 初始化 BTC 和 ETH（真正使用！）
	btcWallet := btc.NewBTCWallet(hdWallet)
	ethWallet := eth.NewETHWallet(hdWallet)

	// 测试生成地址（让变量真正被使用）
	btcResp, _ := btcWallet.GenerateDepositAddress(context.Background(), "testuser001", types.ChainBTC)
	ethResp, _ := ethWallet.GenerateDepositAddress(context.Background(), "testuser001", types.ChainETH)

	log.Printf("✅ BTC 测试地址: %s", btcResp.Address)
	log.Printf("✅ ETH 测试地址: %s", ethResp.Address)

	log.Println("🚀 Wallet Core 服务启动中...")
	log.Println("📡 API 服务已启动: http://localhost:2113")

	// 创建 HTTP Server（支持优雅关闭）
	server := &http.Server{Addr: ":2113"}

	// 注册路由（后面会补全）
	http.HandleFunc("/api/v1/address", func(w http.ResponseWriter, r *http.Request) {
		log.Println("收到地址生成请求")
	})

	http.Handle("/metrics", promhttp.Handler())

	// 启动服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// 优雅关闭（真正使用 ctx）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⛔ 正在优雅关闭 Wallet Core 服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Shutdown 失败:", err)
	}

	log.Println("✅ Wallet Core 已安全关闭")
}