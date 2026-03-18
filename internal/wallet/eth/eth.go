package eth

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/Ixecd/web3-blitz/internal/db"
	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ETHWallet struct {
	hdWallet *core.HDWallet
	rpc      *ethclient.Client // 新增：真实 Geth 客户端
	registry *types.AddressRegistry
	queries  *db.Queries
	hotKey   *ecdsa.PrivateKey // 热钱包私钥，用于签名提币交易
}

// NewETHWallet hotKeyHex 是热钱包私钥的 hex 字符串（不含 0x 前缀）
// geth dev 模式下可从 geth 日志拿到预置账户私钥
func NewETHWallet(hd *core.HDWallet, rpc *ethclient.Client, registry *types.AddressRegistry, queries *db.Queries, hotKeyHex string) *ETHWallet {
	var hotKey *ecdsa.PrivateKey
	if hotKeyHex != "" {
		keyBytes, err := hex.DecodeString(strings.TrimPrefix(hotKeyHex, "0x"))
		if err == nil {
			hotKey, err = crypto.ToECDSA(keyBytes)
			if err != nil {
				log.Printf("[WARN] ETH热钱包私钥解析失败: %v", err)
			}
		}
	}
	return &ETHWallet{hdWallet: hd, rpc: rpc, registry: registry, queries: queries, hotKey: hotKey}
}

// GenerateDepositAddress 生成真实 ETH 充值地址（BIP44）
func (w *ETHWallet) GenerateDepositAddress(ctx context.Context, userID string, chain types.Chain) (types.AddressResponse, error) {
	// 使用 userID 生成 index（简单哈希转 uint32）
	index := uint32(0)
	for _, c := range userID {
		index = index*31 + uint32(c)
	}

	path := fmt.Sprintf("m/44'/60'/0'/0/%d", index)

	// 真实派生
	childKey, err := w.hdWallet.DerivePath(path)
	if err != nil {
		return types.AddressResponse{}, err
	}

	// ETH 地址生成（私钥 → 公钥 → 地址）
	privKeyBytes := childKey.Key
	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return types.AddressResponse{}, err
	}

	address := crypto.PubkeyToAddress(privKey.PublicKey)

	w.registry.Register(address.Hex(), userID)

	err = w.queries.CreateDepositAddress(ctx, db.CreateDepositAddressParams{
		UserID:  userID,
		Address: address.Hex(),
		Chain:   string(chain),
		Path:    path,
	})
	if err != nil {
		log.Printf("[WARN] ETH写入deposit_address失败: %v", err)
	}

	return types.AddressResponse{
		Address: address.Hex(),
		Path:    path,
		UserID:  userID,
	}, nil
}

// GetBalance 真实查询 ETH 余额
func (w *ETHWallet) GetBalance(ctx context.Context, address string, chain types.Chain) (types.BalanceResponse, error) {
	addr := common.HexToAddress(address)
	bal, err := w.rpc.BalanceAt(ctx, addr, nil)
	if err != nil {
		return types.BalanceResponse{}, err
	}

	// wei 转 ETH
	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(bal), new(big.Float).SetInt64(1e18))

	f, _ := ethBalance.Float64()

	return types.BalanceResponse{
		Address: address,
		Balance: f,
		Chain:   chain,
	}, nil
}
