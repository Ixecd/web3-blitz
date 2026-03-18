package eth

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/Ixecd/web3-blitz/internal/config"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/ethereum/go-ethereum/common"
)

type ETHDepositWatcher struct {
	rpc        *config.ETHRPCHolder
	registry   *types.AddressRegistry
	deposits   chan types.DepositRecord
	lastHeight uint64
}

func NewETHDepositWatcher(rpc *config.ETHRPCHolder, registry *types.AddressRegistry) *ETHDepositWatcher {
	return &ETHDepositWatcher{
		rpc:      rpc,
		registry: registry,
		deposits: make(chan types.DepositRecord, 100),
	}
}

func (w *ETHDepositWatcher) Deposits() <-chan types.DepositRecord {
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
		w.processBlock(ctx, h)
	}
	w.lastHeight = currentHeight
}

func (w *ETHDepositWatcher) processBlock(ctx context.Context, height uint64) {
	block, err := w.rpc.Get().BlockByNumber(ctx, new(big.Int).SetUint64(height))
	if err != nil {
		log.Printf("[ERROR] ETH BlockByNumber(%d): %v", height, err)
		return
	}

	for _, tx := range block.Transactions() {
		// 只处理有接收方的交易（排除合约创建）
		if tx.To() == nil {
			continue
		}

		toAddr := tx.To().Hex()

		userID, ok := w.registry.Lookup(toAddr)
		if !ok {
			// 尝试小写匹配
			userID, ok = w.registry.Lookup(common.HexToAddress(toAddr).Hex())
			if !ok {
				continue
			}
		}

		// wei转ETH
		amount := new(big.Float).Quo(
			new(big.Float).SetInt(tx.Value()),
			new(big.Float).SetFloat64(1e18),
		)
		amountFloat, _ := amount.Float64()

		record := types.DepositRecord{
			TxID:      tx.Hash().Hex(),
			Address:   toAddr,
			UserID:    userID,
			Amount:    amountFloat,
			Height:    int64(height),
			Confirmed: false,
			Chain:     types.ChainETH,
		}

		log.Printf("💰 ETH检测到充值! userID=%s address=%s amount=%f txid=%s",
			userID, toAddr, amountFloat, tx.Hash().Hex())

		select {
		case w.deposits <- record:
		default:
			log.Printf("[WARN] ETH deposits channel 满了，丢弃: %s", tx.Hash().Hex())
		}
	}
}
