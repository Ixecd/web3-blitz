package config

import (
	"sync"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/ethereum/go-ethereum/ethclient"
)

// RPCHolder 线程安全的 RPC 客户端持有者
// 所有组件通过 Get 方法获取当前客户端，配置变更时只需调用 Set
type BTCRPCHolder struct {
	mu  sync.RWMutex
	rpc *rpcclient.Client
}

func NewBTCRPCHolder(rpc *rpcclient.Client) *BTCRPCHolder {
	return &BTCRPCHolder{rpc: rpc}
}

func (h *BTCRPCHolder) Get() *rpcclient.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rpc
}

func (h *BTCRPCHolder) Set(rpc *rpcclient.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rpc = rpc
}

type ETHRPCHolder struct {
	mu  sync.RWMutex
	rpc *ethclient.Client
}

func NewETHRPCHolder(rpc *ethclient.Client) *ETHRPCHolder {
	return &ETHRPCHolder{rpc: rpc}
}

func (h *ETHRPCHolder) Get() *ethclient.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rpc
}

func (h *ETHRPCHolder) Set(rpc *ethclient.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rpc = rpc
}