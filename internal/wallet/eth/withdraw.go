package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type WithdrawResult struct {
	TxID string
	Fee  float64
}

// Withdraw 用热钱包私钥签名并广播 ETH 提币交易
func (w *ETHWallet) Withdraw(ctx context.Context, toAddress string, amount float64) (WithdrawResult, error) {
	if w.hotKey == nil {
		return WithdrawResult{}, fmt.Errorf("ETH 热钱包未配置，请设置 ETH_HOT_WALLET_KEY 环境变量")
	}

	fromAddr := crypto.PubkeyToAddress(w.hotKey.PublicKey)
	toAddr := common.HexToAddress(toAddress)

	// float64 ETH → wei (big.Int)
	amountWei, _ := new(big.Float).
		Mul(new(big.Float).SetFloat64(amount), new(big.Float).SetFloat64(1e18)).
		Int(nil)

	nonce, err := w.rpc.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return WithdrawResult{}, fmt.Errorf("获取 nonce 失败: %w", err)
	}

	gasPrice, err := w.rpc.SuggestGasPrice(ctx)
	if err != nil {
		return WithdrawResult{}, fmt.Errorf("获取 gasPrice 失败: %w", err)
	}

	chainID, err := w.rpc.ChainID(ctx)
	if err != nil {
		return WithdrawResult{}, fmt.Errorf("获取 chainID 失败: %w", err)
	}

	const gasLimit = uint64(21000)
	tx := types.NewTransaction(nonce, toAddr, amountWei, gasLimit, gasPrice, nil)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), w.hotKey)
	if err != nil {
		return WithdrawResult{}, fmt.Errorf("交易签名失败: %w", err)
	}

	if err := w.rpc.SendTransaction(ctx, signedTx); err != nil {
		return WithdrawResult{}, fmt.Errorf("广播交易失败: %w", err)
	}

	// fee = gasLimit * gasPrice (wei → ETH)
	feeWei := new(big.Int).Mul(big.NewInt(int64(gasLimit)), gasPrice)
	fee, _ := new(big.Float).Quo(new(big.Float).SetInt(feeWei), new(big.Float).SetFloat64(1e18)).Float64()

	return WithdrawResult{TxID: signedTx.Hash().Hex(), Fee: fee}, nil
}