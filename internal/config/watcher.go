package config

import (
	"context"
	"log"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	KeyBTCRPCHost = "/blitz/config/btc_rpc_host"
	KeyETHRPCHost = "/blitz/config/eth_rpc_host"
)

type ConfigWatcher struct {
	etcd       *clientv3.Client
	btcHolder  *BTCRPCHolder
	ethHolder  *ETHRPCHolder
	btcUser    string
	btcPass    string
}

func NewConfigWatcher(
	etcd *clientv3.Client,
	btcHolder *BTCRPCHolder,
	ethHolder *ETHRPCHolder,
	btcUser, btcPass string,
) *ConfigWatcher {
	return &ConfigWatcher{
		etcd:      etcd,
		btcHolder: btcHolder,
		ethHolder: ethHolder,
		btcUser:   btcUser,
		btcPass:   btcPass,
	}
}

func (w *ConfigWatcher) Start(ctx context.Context) {
	log.Println("🔧 ConfigWatcher 已启动，监听配置变更...")

	watchCh := w.etcd.Watch(ctx, "/blitz/config/", clientv3.WithPrefix())

	for {
		select {
		case <-ctx.Done():
			log.Println("⛔ ConfigWatcher 已停止")
			return
		case resp, ok := <-watchCh:
			if !ok {
				return
			}
			for _, ev := range resp.Events {
				w.handleEvent(ctx, ev)
			}
		}
	}
}

func (w *ConfigWatcher) handleEvent(ctx context.Context, ev *clientv3.Event) {
	key := string(ev.Kv.Key)
	val := string(ev.Kv.Value)

	switch key {
	case KeyBTCRPCHost:
		log.Printf("🔧 检测到 BTC_RPC_HOST 变更: %s", val)
		cfg := &rpcclient.ConnConfig{
			Host:         val,
			User:         w.btcUser,
			Pass:         w.btcPass,
			HTTPPostMode: true,
			DisableTLS:   true,
		}
		newRPC, err := rpcclient.New(cfg, nil)
		if err != nil {
			log.Printf("[ERROR] BTC RPC 重连失败: %v", err)
			return
		}
		w.btcHolder.Set(newRPC)
		log.Printf("✅ BTC RPC 已热更新: %s", val)

	case KeyETHRPCHost:
		log.Printf("🔧 检测到 ETH_RPC_HOST 变更: %s", val)
		newRPC, err := ethclient.DialContext(ctx, val)
		if err != nil {
			log.Printf("[ERROR] ETH RPC 重连失败: %v", err)
			return
		}
		w.ethHolder.Set(newRPC)
		log.Printf("✅ ETH RPC 已热更新: %s", val)
	}
}