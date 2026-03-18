package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ixecd/web3-blitz/internal/api"
	"github.com/Ixecd/web3-blitz/internal/config"
	"github.com/Ixecd/web3-blitz/internal/db"
	"github.com/Ixecd/web3-blitz/internal/lock"
	"github.com/Ixecd/web3-blitz/internal/wallet/btc"
	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/eth"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	// 生命周期控制，最先创建
	ctx, cancel := context.WithCancel(context.Background())

	// 密钥体系，其他所有钱包操作的根基
	hdWallet, err := core.NewHDWallet()
	if err != nil {
		log.Fatal("HDWallet 初始化失败:", err)
	}

	// 持久层，基础设施
	database, err := db.NewDB("./blitz.db")
	if err != nil {
		log.Fatal("DB初始化失败:", err)
	}

	// DB操作句柄
	queries := db.New(database)
	log.Println("✅ 数据库已连接")

	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		etcdEndpoints = "localhost:2379"
	}
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdEndpoints},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal("etcd 连接失败:", err)
	}
	defer etcdClient.Close()
	log.Println("✅ etcd 已连接")

	locker := lock.NewDistributedLock(etcdClient, 30)

	// 先创建空的registry
	registry := types.NewAddressRegistry()

	// 用queries填充registry，服务重启不丢状态
	addrs, err := queries.ListAllDepositAddresses(context.Background())
	if err != nil {
		log.Printf("[WARN] 恢复registry失败: %v", err)
	} else {
		for _, a := range addrs {
			registry.Register(a.Address, a.UserID)
		}
		log.Printf("✅ 从DB恢复了 %d 个充值地址", len(addrs))
	}

	btcRPCHost := os.Getenv("BTC_RPC_HOST")
	if btcRPCHost == "" {
		btcRPCHost = "localhost:18443/wallet/blitz_wallet"
	}

	ethRPCHost := os.Getenv("ETH_RPC_HOST")
	if ethRPCHost == "" {
		ethRPCHost = "http://localhost:8545"
	}

	// 创建真实 RPC 客户端，外部连接
	btcCfg := &rpcclient.ConnConfig{
		Host:         btcRPCHost,
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	btcRPC, _ := rpcclient.New(btcCfg, nil)
	ethRPC, _ := ethclient.Dial(ethRPCHost)

	// 从环境变量读热钱包私钥
	hotKeyHex := os.Getenv("ETH_HOT_WALLET_KEY")

	// 创建 Holder
	btcRPCHolder := config.NewBTCRPCHolder(btcRPC)
	ethRPCHolder := config.NewETHRPCHolder(ethRPC)

	// ConfigWatcher
	configWatcher := config.NewConfigWatcher(etcdClient, btcRPCHolder, ethRPCHolder, "user", "pass")
	go configWatcher.Start(ctx)

	// 组件注入 Holder 而非直接的 rpc 客户端
	btcWallet := btc.NewBTCWallet(hdWallet, btcRPCHolder, registry, queries)
	ethWallet := eth.NewETHWallet(hdWallet, ethRPCHolder, registry, queries, hotKeyHex)
	watcher := btc.NewDepositWatcher(btcRPCHolder, registry)
	ethWatcher := eth.NewETHDepositWatcher(ethRPCHolder, registry)
	confirmChecker := core.NewConfirmChecker(queries, btcRPCHolder, ethRPCHolder, etcdClient)

	log.Println("🚀 Wallet Core 服务已启动（真实 RPC 已连接）")

	// h := api.NewHandler(btcWallet, ethWallet, queries, redisClient)
	h := api.NewHandler(btcWallet, ethWallet, queries, locker)
	mux := api.NewMux(h)

	go confirmChecker.Start(ctx)
	// 开始扫块
	go watcher.Start(ctx)
	go ethWatcher.Start(ctx)

	// 抽成函数
	consumeDeposits := func(ch <-chan types.DepositRecord, chainName string) {
		for deposit := range ch {
			log.Printf("📥 %s入账处理: %+v", chainName, deposit)
			confirmed := int32(0)
			if deposit.Confirmed {
				confirmed = 1
			}
			err := queries.CreateDeposit(context.Background(), db.CreateDepositParams{
				TxID:      deposit.TxID,
				Address:   deposit.Address,
				UserID:    deposit.UserID,
				Amount:    fmt.Sprintf("%.8f", deposit.Amount),
				Height:    int64(deposit.Height),
				Confirmed: confirmed,
				Chain:     string(deposit.Chain),
			})
			if err != nil {
				log.Printf("[ERROR] %s写入deposit失败: %v", chainName, err)
			} else {
				log.Printf("✅ %s deposit已写入DB: txid=%s", chainName, deposit.TxID)
			}
		}
	}

	go consumeDeposits(watcher.Deposits(), "BTC")
	go consumeDeposits(ethWatcher.Deposits(), "ETH")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	port := os.Getenv("PORT")
	if port == "" {
		port = "2113"
	}
	srv := &http.Server{Addr: ":" + port, Handler: mux}

	// 信号处理goroutine
	go func() {
		<-sig
		log.Println("⛔ 收到关闭信号，正在优雅关闭...")
		srv.Shutdown(context.Background())
		cancel()
	}()

	log.Println("📡 API 服务已启动: http://localhost:2113")
	log.Println(`测试余额: curl "http://localhost:2113/api/v1/balance?address=你的地址&chain=btc"`)

	// 开始接受请求
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	// 等待退出信号
	<-ctx.Done()
}
