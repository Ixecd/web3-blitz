package btc

import (
	"context"
	"fmt"

	"github.com/Ixecd/web3-blitz/internal/wallet/core"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

type BTCWallet struct {
	hdWallet *core.HDWallet
}

func NewBTCWallet(hd *core.HDWallet) *BTCWallet {
	return &BTCWallet{hdWallet: hd}
}

func (w *BTCWallet) GenerateDepositAddress(ctx context.Context, userID string, chain types.Chain) (types.AddressResponse, error) {
	index := uint32(0)
	child, err := w.hdWallet.MasterKey.NewChildKey(index)
	if err != nil {
		return types.AddressResponse{}, err
	}

	addr, err := btcutil.NewAddressWitnessPubKeyHash(
		btcutil.Hash160(child.PublicKey().Key),
		&chaincfg.MainNetParams,
	)
	if err != nil {
		return types.AddressResponse{}, err
	}

	path := fmt.Sprintf("m/44'/0'/0'/0/%d", index)

	return types.AddressResponse{
		Address: addr.EncodeAddress(),
		Path:    path,
		UserID:  userID,
	}, nil
}

func (w *BTCWallet) GetBalance(ctx context.Context, address string, chain types.Chain) (types.BalanceResponse, error) {
	return types.BalanceResponse{
		Address: address,
		Balance: 0.0,
		Chain:   chain,
	}, nil
}