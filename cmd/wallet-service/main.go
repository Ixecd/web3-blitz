package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ixecd/web3-blitz/internal/wallet/btc"
	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/eth"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"
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

	ctx, cancel := context.WithCancel(context.Background())

	// ✅ 正确调用（无参数，从环境变量读取）
	hdWallet, err := core.NewHDWallet()
	if err != nil {
		log.Fatal("HDWallet 初始化失败:", err)
	}

	// 创建真实 RPC 客户端
	btcCfg := &rpcclient.ConnConfig{
		Host:         "localhost:18443/wallet/blitz_wallet",
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	btcRPC, _ := rpcclient.New(btcCfg, nil)

	ethRPC, _ := ethclient.Dial("http://localhost:8545")

	registry := btc.NewAddressRegistry()

	btcWallet := btc.NewBTCWallet(hdWallet, btcRPC, registry)
	ethWallet := eth.NewETHWallet(hdWallet, ethRPC)

	log.Println("🚀 Wallet Core 服务已启动（真实 RPC 已连接）")

	watcher := btc.NewDepositWatcher(btcRPC, registry)

	// === HTTP API ===
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	// 地址生成接口
	mux.HandleFunc("/api/v1/address", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserID string      `json:"user_id"`
			Chain  types.Chain `json:"chain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "参数错误", http.StatusBadRequest)
			return
		}

		if req.UserID == "" {
			http.Error(w, "user_id 不能为空", http.StatusBadRequest)
			return
		}

		var resp types.AddressResponse
		var genErr error

		if req.Chain == types.ChainBTC {
			resp, genErr = btcWallet.GenerateDepositAddress(r.Context(), req.UserID, req.Chain)
		} else if req.Chain == types.ChainETH {
			resp, genErr = ethWallet.GenerateDepositAddress(r.Context(), req.UserID, req.Chain)
		} else {
			http.Error(w, "不支持的链", http.StatusBadRequest)
			return
		}

		if genErr != nil {
			http.Error(w, genErr.Error(), http.StatusInternalServerError)
			return
		}

		walletAddressesTotal.Inc()
		json.NewEncoder(w).Encode(resp)
	})

	go watcher.Start(ctx)

	go func() {
		for deposit := range watcher.Deposits() {
			log.Printf("📥 入账处理: %+v", deposit)
			// TODO: 写数据库
		}
	}()

	// 余额查询接口（真实查询）
	mux.HandleFunc("/api/v1/balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "只支持 GET", http.StatusMethodNotAllowed)
			return
		}

		address := r.URL.Query().Get("address")
		chainStr := r.URL.Query().Get("chain")

		if address == "" || chainStr == "" {
			http.Error(w, "缺少 address 或 chain 参数", http.StatusBadRequest)
			return
		}

		var resp types.BalanceResponse
		var err error

		if chainStr == "btc" {
			resp, err = btcWallet.GetBalance(r.Context(), address, types.ChainBTC)
		} else if chainStr == "eth" {
			resp, err = ethWallet.GetBalance(r.Context(), address, types.ChainETH)
		} else {
			http.Error(w, "不支持的链", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(resp)
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{Addr: ":2113", Handler: mux}

	go func() {
		<-sig
		log.Println("⛔ 收到关闭信号，正在优雅关闭...")
		srv.Shutdown(context.Background())
		cancel()
	}()

	log.Println("📡 API 服务已启动: http://localhost:2113")
	log.Println(`测试余额: curl "http://localhost:2113/api/v1/balance?address=你的地址&chain=btc"`)
	
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-ctx.Done()
}
