package eth

import (
	"context"
	"fmt"

	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type ETHWallet struct {
	hdWallet *core.HDWallet
}

func NewETHWallet(hd *core.HDWallet) *ETHWallet {
	return &ETHWallet{hdWallet: hd}
}

// GenerateDepositAddress 生成真实 ETH 充值地址
func (w *ETHWallet) GenerateDepositAddress(ctx context.Context, userID string, chain types.Chain) (types.AddressResponse, error) {
	index := uint32(0)
	child, err := w.hdWallet.MasterKey.NewChildKey(index)
	if err != nil {
		return types.AddressResponse{}, err
	}

	privKey, err := crypto.ToECDSA(child.Key)
	if err != nil {
		return types.AddressResponse{}, err
	}

	address := crypto.PubkeyToAddress(privKey.PublicKey)

	path := fmt.Sprintf("m/44'/60'/0'/0/%d", index)

	return types.AddressResponse{
		Address: address.Hex(),
		Path:    path,
		UserID:  userID,
	}, nil
}

func (w *ETHWallet) GetBalance(ctx context.Context, address string, chain types.Chain) (types.BalanceResponse, error) {
	return types.BalanceResponse{
		Address: address,
		Balance: 0.0,
		Chain:   chain,
	}, nil
}