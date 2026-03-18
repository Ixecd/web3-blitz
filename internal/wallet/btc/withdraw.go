package btc

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
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

	txHash, err := w.rpc.SendToAddress(addr, satoshis)
	if err != nil {
		return WithdrawResult{}, fmt.Errorf("BTC 广播失败: %w", err)
	}

	return WithdrawResult{TxID: txHash.String(), Fee: 0}, nil
}