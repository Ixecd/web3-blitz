package btc

import (
	"context"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type WithdrawResult struct {
	TxID string
	Fee  float64
}

// Withdraw 通过 bitcoind 热钱包签名广播提币交易
func (w *BTCWallet) Withdraw(ctx context.Context, toAddress string, amount float64) (WithdrawResult, error) {
	addr, err := btcutil.DecodeAddress(toAddress, &chaincfg.RegressionNetParams)
	if err != nil {
		return WithdrawResult{}, fmt.Errorf("BTC 地址解析失败: %w", err)
	}

	satoshis, err := btcutil.NewAmount(amount)
	if err != nil {
		return WithdrawResult{}, fmt.Errorf("金额转换失败: %w", err)
	}

	txHash, err := w.rpcHolder.Get().SendToAddress(addr, satoshis)
	if err != nil {
		return WithdrawResult{}, fmt.Errorf("BTC 广播失败: %w", err)
	}

	fee := w.queryFee(txHash)

	return WithdrawResult{TxID: txHash.String(), Fee: fee}, nil
}

// queryFee 通过 GetTransaction 回查实际扣除的手续费
func (w *BTCWallet) queryFee(txHash *chainhash.Hash) float64 {
	tx, err := w.rpcHolder.Get().GetTransaction(txHash)
	if err != nil {
		log.Printf("[WARN] 回查 BTC fee 失败 txid=%s: %v", txHash.String(), err)
		return 0
	}
	// bitcoind 返回的 fee 是负数（支出），取绝对值
	fee := tx.Fee
	if fee < 0 {
		fee = -fee
	}
	return fee
}
