package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ETHWallet struct {
	hdWallet *core.HDWallet
	rpc      *ethclient.Client   // 新增：真实 Geth 客户端
}

func NewETHWallet(hd *core.HDWallet, rpc *ethclient.Client) *ETHWallet {
	return &ETHWallet{hdWallet: hd, rpc: rpc}
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