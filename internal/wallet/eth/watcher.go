package eth

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/Ixecd/web3-blitz/internal/config"
	wallettypes "github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/ethereum/go-ethereum/common"
)

type ETHDepositWatcher struct {
	rpc        *config.ETHRPCHolder
	registry   *wallettypes.AddressRegistry
	deposits   chan wallettypes.DepositRecord
	lastHeight uint64
	lastHash   common.Hash
}

func NewETHDepositWatcher(rpc *config.ETHRPCHolder, registry *wallettypes.AddressRegistry) *ETHDepositWatcher {
	return &ETHDepositWatcher{
		rpc:      rpc,
		registry: registry,
		deposits: make(chan wallettypes.DepositRecord, 100),
	}
}

func (w *ETHDepositWatcher) Deposits() <-chan wallettypes.DepositRecord {
	return w.deposits
}

func (w *ETHDepositWatcher) Start(ctx context.Context) {
	log.Println("🔍 ETH Deposit Watcher 已启动")

	header, err := w.rpc.Get().HeaderByNumber(ctx, nil)
	if err != nil {
		log.Printf("[ERROR] 获取ETH最新块高失败: %v", err)
		return
	}
	w.lastHeight = header.Number.Uint64()
	w.lastHash = header.Hash()
	log.Printf("📦 ETH从块高 %d 开始监听", w.lastHeight)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("⛔ ETH Deposit Watcher 已停止")
			return
		case <-ticker.C:
			w.scanNewBlocks(ctx)
		}
	}
}

func (w *ETHDepositWatcher) scanNewBlocks(ctx context.Context) {
	header, err := w.rpc.Get().HeaderByNumber(ctx, nil)
	if err != nil {
		log.Printf("[ERROR] ETH scanNewBlocks: %v", err)
		return
	}

	currentHeight := header.Number.Uint64()
	if currentHeight <= w.lastHeight {
		return
	}

	for h := w.lastHeight + 1; h <= currentHeight; h++ {
		ok := w.processBlock(ctx, h)
		if !ok {
			// 处理失败，停止本轮扫块，下次重试
			return
		}
	}
	w.lastHeight = currentHeight
}

// processBlock 处理单个块，返回是否成功
func (w *ETHDepositWatcher) processBlock(ctx context.Context, height uint64) bool {
	block, err := w.rpc.Get().BlockByNumber(ctx, new(big.Int).SetUint64(height))
	if err != nil {
		log.Printf("[ERROR] ETH BlockByNumber(%d): %v", height, err)
		return false
	}

	// reorg 检测：当前块的 ParentHash 应该等于上一个已处理块的 Hash
	if height > 0 && w.lastHash != (common.Hash{}) {
		if block.ParentHash() != w.lastHash {
			log.Printf("⚠️  [REORG] ETH 检测到链重组！块高 %d，期望 parentHash=%s，实际 parentHash=%s",
				height, w.lastHash.Hex(), block.ParentHash().Hex())
			log.Printf("⚠️  [REORG] 回退到块高 %d 重新扫描", height-1)

			// 回退：重新获取上一块，修正 lastHash
			prevBlock, err := w.rpc.Get().BlockByNumber(ctx, new(big.Int).SetUint64(height-1))
			if err != nil {
				log.Printf("[ERROR] REORG 回退获取块失败: %v", err)
				return false
			}
			w.lastHash = prevBlock.Hash()
			w.lastHeight = height - 1

			log.Printf("⚠️  [REORG] 已回退，新 lastHeight=%d lastHash=%s，下次轮询重新扫描",
				w.lastHeight, w.lastHash.Hex())
			return false
		}
	}

	// 正常处理交易
	for _, tx := range block.Transactions() {
		if tx.To() == nil {
			continue
		}

		toAddr := tx.To().Hex()
		userID, ok := w.registry.Lookup(toAddr)
		if !ok {
			userID, ok = w.registry.Lookup(common.HexToAddress(toAddr).Hex())
			if !ok {
				continue
			}
		}

		amount := new(big.Float).Quo(
			new(big.Float).SetInt(tx.Value()),
			new(big.Float).SetFloat64(1e18),
		)
		amountFloat, _ := amount.Float64()

		record := wallettypes.DepositRecord{
			TxID:      tx.Hash().Hex(),
			Address:   toAddr,
			UserID:    userID,
			Amount:    amountFloat,
			Height:    int64(height),
			Confirmed: false,
			Chain:     wallettypes.ChainETH,
		}

		log.Printf("💰 ETH检测到充值! userID=%s address=%s amount=%f txid=%s",
			userID, toAddr, amountFloat, tx.Hash().Hex())

		select {
		case w.deposits <- record:
		default:
			log.Printf("[WARN] ETH deposits channel 满了，丢弃: %s", tx.Hash().Hex())
		}
	}

	// 更新 lastHash
	w.lastHash = block.Hash()
	return true
}
