package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/Ixecd/web3-blitz/internal/api"
	"github.com/Ixecd/web3-blitz/internal/config"
	"github.com/Ixecd/web3-blitz/internal/db"
	"github.com/Ixecd/web3-blitz/internal/email"
	"github.com/Ixecd/web3-blitz/internal/lock"
	"github.com/Ixecd/web3-blitz/internal/logger"
	"github.com/Ixecd/web3-blitz/internal/metrics"
	"github.com/Ixecd/web3-blitz/internal/wallet/btc"
	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/eth"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	godotenv.Load()

	// 最先初始化日志，后续所有 slog 调用才能正确路由
	logger.Init()

	// 生命周期控制
	ctx, cancel := context.WithCancel(context.Background())
	metrics.Init()

	// 密钥体系，其他所有钱包操作的根基
	hdWallet, err := core.NewHDWallet()
	if err != nil {
		slog.Error("HDWallet 初始化失败", "err", err)
		os.Exit(1)
	}

	// 持久层
	database, err := db.NewDB()
	if err != nil {
		slog.Error("DB 初始化失败", "err", err)
		os.Exit(1)
	}

	queries := db.New(database)

	// etcd
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379"
	}
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdEndpoints},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		slog.Error("etcd 连接失败", "err", err)
		os.Exit(1)
	}
	defer etcdClient.Close()
	slog.Info("etcd 已连接")

	locker := lock.NewDistributedLock(etcdClient, 30)

	// 从 DB 恢复充值地址注册表，服务重启不丢状态
	registry := types.NewAddressRegistry()
	addrs, err := queries.ListAllDepositAddresses(context.Background())
	if err != nil {
		slog.Warn("恢复 registry 失败", "err", err)
	} else {
		for _, a := range addrs {
			registry.Register(a.Address, a.UserID)
		}
		slog.Info("从 DB 恢复充值地址", "count", len(addrs))
	}

	// RPC 配置
	btcRPCHost := os.Getenv("BTC_RPC_HOST")
	if btcRPCHost == "" {
		btcRPCHost = "localhost:18443/wallet/blitz_wallet"
	}
	ethRPCHost := os.Getenv("ETH_RPC_HOST")
	if ethRPCHost == "" {
		ethRPCHost = "http://localhost:8545"
	}

	btcCfg := &rpcclient.ConnConfig{
		Host:         btcRPCHost,
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	btcRPC, _ := rpcclient.New(btcCfg, nil)
	ethRPC, _ := ethclient.Dial(ethRPCHost)

	hotKeyHex := os.Getenv("ETH_HOT_WALLET_KEY")

	btcRPCHolder := config.NewBTCRPCHolder(btcRPC)
	ethRPCHolder := config.NewETHRPCHolder(ethRPC)

	configWatcher := config.NewConfigWatcher(etcdClient, btcRPCHolder, ethRPCHolder, "user", "pass")
	go configWatcher.Start(ctx)

	btcWallet := btc.NewBTCWallet(hdWallet, btcRPCHolder, registry, queries)
	ethWallet := eth.NewETHWallet(hdWallet, ethRPCHolder, registry, queries, hotKeyHex)
	watcher := btc.NewDepositWatcher(btcRPCHolder, registry)
	ethWatcher := eth.NewETHDepositWatcher(ethRPCHolder, registry)
	confirmChecker := core.NewConfirmChecker(queries, btcRPCHolder, ethRPCHolder, etcdClient)

	slog.Info("Wallet Core 服务已启动")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
		slog.Warn("使用测试 JWT secret，生产环境请务必设置 JWT_SECRET 环境变量")
	}

	mailer := email.NewMailer()

	h := api.NewHandler(btcWallet, ethWallet, queries, database, locker, jwtSecret, mailer)
	mux := api.NewMux(h, jwtSecret, queries)

	go confirmChecker.Start(ctx)
	go watcher.Start(ctx)
	go ethWatcher.Start(ctx)

	consumeDeposits := func(ch <-chan types.DepositRecord, chainName string) {
		for deposit := range ch {
			slog.Info("入账处理", "chain", chainName, "tx_id", deposit.TxID, "amount", deposit.Amount)

			confirmed := int32(0)
			if deposit.Confirmed {
				confirmed = 1
			}
			params := db.CreateDepositParams{
				TxID:      deposit.TxID,
				Address:   deposit.Address,
				UserID:    deposit.UserID,
				Amount:    fmt.Sprintf("%.8f", deposit.Amount),
				Height:    int64(deposit.Height),
				Confirmed: confirmed,
				Chain:     string(deposit.Chain),
			}

			const maxRetries = 3
			var lastErr error
			for i := range maxRetries {
				lastErr = queries.CreateDeposit(context.Background(), params)
				if lastErr == nil {
					metrics.DepositTotal.WithLabelValues(string(deposit.Chain), "detected").Inc()
					metrics.DepositAmount.WithLabelValues(string(deposit.Chain)).Add(deposit.Amount)
					slog.Info("deposit 已写入 DB", "chain", chainName, "tx_id", deposit.TxID)
					break
				}
				slog.Warn("写入 deposit 失败，重试中", "chain", chainName, "attempt", i+1, "err", lastErr)
				time.Sleep(time.Duration(i+1) * time.Second)
			}

			if lastErr != nil {
				metrics.DeadLetterTotal.WithLabelValues(chainName + "_deposit").Inc()
				slog.Error("deposit 写入最终失败，写入死信队列", "chain", chainName, "tx_id", deposit.TxID, "err", lastErr)
				payload, _ := json.Marshal(params)
				_ = queries.CreateDeadLetter(context.Background(), db.CreateDeadLetterParams{
					Type:    chainName + "_deposit",
					Payload: payload,
					Error:   lastErr.Error(),
					Retries: maxRetries,
				})
			}
		}
	}

	go consumeDeposits(watcher.Deposits(), "BTC")
	go consumeDeposits(ethWatcher.Deposits(), "ETH")

	port := os.Getenv("PORT")
	if port == "" {
		port = "2113"
	}
	srv := &http.Server{Addr: ":" + port, Handler: mux}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		slog.Info("收到关闭信号，正在优雅关闭...")
		srv.Shutdown(context.Background())
		cancel()
	}()

	slog.Info("API 服务已启动", "addr", "http://localhost:"+port)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP 服务异常退出", "err", err)
		os.Exit(1)
	}

	<-ctx.Done()
}
