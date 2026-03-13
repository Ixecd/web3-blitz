package core

import "github.com/tyler-smith/go-bip32"

type HDWallet struct {
	MasterKey *bip32.Key   // 大写 M，已导出
}

func NewHDWallet(seed []byte) (*HDWallet, error) {
	master, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, err
	}
	return &HDWallet{MasterKey: master}, nil
}