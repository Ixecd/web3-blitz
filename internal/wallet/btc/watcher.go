package btc

import (
	"context"
	"log"
	"time"

	"github.com/Ixecd/web3-blitz/internal/config"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

// DepositWatcher 监听BTC充值
type DepositWatcher struct {
	rpc        *config.BTCRPCHolder
	registry   *types.AddressRegistry
	deposits   chan types.DepositRecord
	lastHeight int64
}

func NewDepositWatcher(rpc *config.BTCRPCHolder, registry *types.AddressRegistry) *DepositWatcher {
	return &DepositWatcher{
		rpc:      rpc,
		registry: registry,
		deposits: make(chan types.DepositRecord, 100),
	}
}

// Deposits 返回充值事件channel，外部消费
func (w *DepositWatcher) Deposits() <-chan types.DepositRecord {
	return w.deposits
}

// Start 启动监听，每3秒扫一次新块
func (w *DepositWatcher) Start(ctx context.Context) {
	log.Println("🔍 Deposit Watcher 已启动")

	// 从当前最新块开始扫
	info, err := w.rpc.Get().GetBlockChainInfo()
	if err != nil {
		log.Printf("[ERROR] 获取链信息失败: %v", err)
		return
	}
	w.lastHeight = int64(info.Blocks)
	log.Printf("📦 从块高 %d 开始监听", w.lastHeight)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("⛔ Deposit Watcher 已停止")
			return
		case <-ticker.C:
			w.scanNewBlocks()
		}
	}
}

func (w *DepositWatcher) scanNewBlocks() {
	info, err := w.rpc.Get().GetBlockChainInfo()
	if err != nil {
		log.Printf("[ERROR] scanNewBlocks: %v", err)
		return
	}

	currentHeight := int64(info.Blocks)
	if currentHeight <= w.lastHeight {
		return
	}

	for h := w.lastHeight + 1; h <= currentHeight; h++ {
		w.processBlock(h)
	}
	w.lastHeight = currentHeight
}

func (w *DepositWatcher) processBlock(height int64) {
	hash, err := w.rpc.Get().GetBlockHash(height)
	if err != nil {
		log.Printf("[ERROR] GetBlockHash(%d): %v", height, err)
		return
	}

	block, err := w.rpc.Get().GetBlockVerboseTx(hash)
	if err != nil {
		log.Printf("[ERROR] GetBlockVerboseTx: %v", err)
		return
	}

	for _, tx := range block.Tx {
		for _, vout := range tx.Vout {
			// 从scriptPubKey拿地址
			if len(vout.ScriptPubKey.Address) == 0 {
				continue
			}

			address := vout.ScriptPubKey.Address

			// 验证地址格式
			_, err := btcutil.DecodeAddress(address, &chaincfg.RegressionNetParams)
			if err != nil {
				continue
			}

			// 查注册表
			userID, ok := w.registry.Lookup(address)
			if !ok {
				continue
			}

			record := types.DepositRecord{
				TxID:      tx.Txid,
				Address:   address,
				UserID:    userID,
				Amount:    vout.Value,
				Height:    height,
				Confirmed: false,
				Chain:     types.ChainBTC,
			}

			log.Printf("💰 检测到充值! userID=%s address=%s amount=%f txid=%s",
				userID, address, vout.Value, tx.Txid)

			select {
			case w.deposits <- record:
			default:
				log.Printf("[WARN] deposits channel 满了，丢弃: %s", tx.Txid)
			}
		}
	}
}
