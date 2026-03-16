package btc

import (
	"context"
	"fmt"

	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
)

type BTCWallet struct {
	hdWallet *core.HDWallet
	rpc      *rpcclient.Client   // 新增：真实 RPC 客户端
	registry *AddressRegistry
}

func NewBTCWallet(hd *core.HDWallet, rpc *rpcclient.Client, registry *AddressRegistry) *BTCWallet {
	return &BTCWallet{hdWallet: hd, rpc: rpc, registry: registry}
}

// GenerateDepositAddress 生成真实 BTC 地址（BIP44）+ 自动导入到 miner 钱包（关键！）
func (w *BTCWallet) GenerateDepositAddress(ctx context.Context, userID string, chain types.Chain) (types.AddressResponse, error) {
	// 使用 userID 生成 index（简单 hash 转 uint32）
	index := uint32(0)
	for _, c := range userID {
		index = index*31 + uint32(c)
	}

	path := fmt.Sprintf("m/44'/0'/0'/0/%d", index)

	// 真实派生
	childKey, err := w.hdWallet.DerivePath(path)
	if err != nil {
		return types.AddressResponse{}, err
	}

	// 生成 bech32 地址（P2WPKH）
	addr, err := btcutil.NewAddressWitnessPubKeyHash(
		btcutil.Hash160(childKey.PublicKey().Key),
		&chaincfg.RegressionNetParams,
	)
	if err != nil {
		return types.AddressResponse{}, err
	}

	addressStr := addr.EncodeAddress()

	// // ✅ 关键修复：使用现代方法导入地址（rescan=false 更快）
	// err = w.rpc.ImportAddressRescan(addressStr, "", false)
	// if err != nil {
	// 	// 如果已经导入过，会报错，但不影响使用
	// 	log.Printf("[WARN] ImportAddress: %v", err)
	// }

	w.registry.Register(addressStr, userID)

	return types.AddressResponse{
		Address: addr.EncodeAddress(),
		Path:    path,
		UserID:  userID,
	}, nil
}

// GetBalance 真实查询 BTC 地址收到的金额（充值地址专用）
func (w *BTCWallet) GetBalance(ctx context.Context, address string, chain types.Chain) (types.BalanceResponse, error) {
	// 1. 把字符串地址解析成 btcutil.Address
	addr, err := btcutil.DecodeAddress(address, &chaincfg.RegressionNetParams)
	if err != nil {
		return types.BalanceResponse{}, fmt.Errorf("地址解析失败 (regtest): %w", err)
	}

	// 2. 查询该地址收到的总金额
	bal, err := w.rpc.GetReceivedByAddress(addr)
	if err != nil {
		return types.BalanceResponse{}, fmt.Errorf("查询 BTC 余额失败: %w", err)
	}

	return types.BalanceResponse{
		Address: address,
		Balance: bal.ToBTC(),
		Chain:   chain,
	}, nil
}