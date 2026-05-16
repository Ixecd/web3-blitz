package btc

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
)

var (
	netParams     *chaincfg.Params
	netParamsOnce sync.Once
)

// NetParams 根据 BTC_NETWORK 环境变量返回比特币网络参数。
//
//	mainnet  → MainNetParams
//	testnet  → TestNet3Params
//	simnet   → SimNetParams
//	regtest  → RegressionNetParams（默认，本地开发）
func NetParams() *chaincfg.Params {
	netParamsOnce.Do(func() {
		raw := strings.ToLower(strings.TrimSpace(os.Getenv("BTC_NETWORK")))
		switch raw {
		case "mainnet":
			netParams = &chaincfg.MainNetParams
		case "testnet", "testnet3":
			netParams = &chaincfg.TestNet3Params
		case "simnet":
			netParams = &chaincfg.SimNetParams
		case "regtest", "":
			netParams = &chaincfg.RegressionNetParams
		default:
			panic(fmt.Sprintf("BTC_NETWORK=%q 不支持，可选: mainnet/testnet/simnet/regtest", raw))
		}
	})
	return netParams
}
