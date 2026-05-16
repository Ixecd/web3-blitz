package sweep

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Ixecd/blitz/internal/audit"
	"github.com/Ixecd/blitz/internal/config"
	"github.com/Ixecd/blitz/internal/metrics"
	btcchain "github.com/Ixecd/blitz/internal/wallet/btc"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Sweeper periodically transfers hot wallet funds to cold storage.
type Sweeper struct {
	btcRPC       *config.BTCRPCHolder
	ethRPC       *config.ETHRPCHolder
	coldAddrBTC  string
	coldAddrETH  string
	keepBTC      float64
	keepETH      float64
	btcFeeBuffer float64
	ethHotKeyHex string
}

// New creates a Sweeper from environment variables.
func New(btcRPC *config.BTCRPCHolder, ethRPC *config.ETHRPCHolder) *Sweeper {
	s := &Sweeper{
		btcRPC:       btcRPC,
		ethRPC:       ethRPC,
		coldAddrBTC:  os.Getenv("COLD_WALLET_BTC"),
		coldAddrETH:  os.Getenv("COLD_WALLET_ETH"),
		keepBTC:      0.01,
		keepETH:      0.1,
		btcFeeBuffer: 0.0001,
		ethHotKeyHex: os.Getenv("ETH_HOT_WALLET_KEY"),
	}

	if v, err := strconv.ParseFloat(os.Getenv("HOT_WALLET_KEEP_BTC"), 64); err == nil {
		s.keepBTC = v
	}
	if v, err := strconv.ParseFloat(os.Getenv("HOT_WALLET_KEEP_ETH"), 64); err == nil {
		s.keepETH = v
	}
	if v, err := strconv.ParseFloat(os.Getenv("SWEEP_BTC_FEE_BUFFER"), 64); err == nil {
		s.btcFeeBuffer = v
	}

	return s
}

// Run executes a single sweep pass for both chains.
func (s *Sweeper) Run(ctx context.Context) {
	slog.Info("Sweep started",
		"cold_btc", s.coldAddrBTC,
		"cold_eth", s.coldAddrETH,
		"keep_btc", fmt.Sprintf("%.8f", s.keepBTC),
		"keep_eth", fmt.Sprintf("%.8f", s.keepETH),
	)

	if s.coldAddrBTC != "" {
		s.sweepBTC(ctx)
	} else {
		slog.Warn("Sweep: COLD_WALLET_BTC not configured, skipping BTC")
	}

	if s.coldAddrETH != "" && s.ethHotKeyHex != "" {
		s.sweepETH(ctx)
	} else {
		if s.coldAddrETH == "" {
			slog.Warn("Sweep: COLD_WALLET_ETH not configured, skipping ETH")
		} else {
			slog.Warn("Sweep: ETH_HOT_WALLET_KEY not configured, skipping ETH")
		}
	}

	slog.Info("Sweep finished")
}

// sweepBTC transfers BTC from the bitcoind hot wallet to cold storage.
func (s *Sweeper) sweepBTC(ctx context.Context) {
	const chain = "btc"
	start := time.Now()
	defer func() {
		slog.Info("Sweep BTC done", "elapsed", time.Since(start).String())
	}()

	balanceBtc, err := s.btcRPC.Get().GetBalance("*")
	if err != nil {
		slog.Error("Sweep BTC: failed to get balance", "err", err)
		metrics.SweepBTCTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, "0", err.Error())
		return
	}

	balance := balanceBtc.ToBTC()
	slog.Info("Sweep BTC: current hot wallet balance", "balance", fmt.Sprintf("%.8f", balance))

	sweepAmount := balance - s.keepBTC - s.btcFeeBuffer
	if sweepAmount <= 0 {
		slog.Info("Sweep BTC: balance below keep threshold, skipping",
			"balance", fmt.Sprintf("%.8f", balance),
			"keep", fmt.Sprintf("%.8f", s.keepBTC))
		metrics.SweepBTCTotal.WithLabelValues("skipped").Inc()
		return
	}

	coldAddr, err := btcutil.DecodeAddress(s.coldAddrBTC, btcchain.NetParams())
	if err != nil {
		slog.Error("Sweep BTC: invalid cold wallet address", "addr", s.coldAddrBTC, "err", err)
		metrics.SweepBTCTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, fmt.Sprintf("%.8f", sweepAmount), "invalid address: "+err.Error())
		return
	}

	amount, err := btcutil.NewAmount(sweepAmount)
	if err != nil {
		slog.Error("Sweep BTC: invalid amount", "amount", sweepAmount, "err", err)
		metrics.SweepBTCTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, fmt.Sprintf("%.8f", sweepAmount), "invalid amount: "+err.Error())
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	txHash, err := s.btcRPC.Get().SendToAddress(coldAddr, amount)
	if err != nil {
		slog.Error("Sweep BTC: broadcast failed", "err", err)
		metrics.SweepBTCTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, fmt.Sprintf("%.8f", sweepAmount), "broadcast: "+err.Error())
		return
	}

	slog.Warn("Sweep BTC: funds transferred to cold storage",
		"amount", fmt.Sprintf("%.8f", sweepAmount),
		"cold_addr", s.coldAddrBTC,
		"tx_id", txHash.String(),
	)
	metrics.SweepBTCTotal.WithLabelValues("success").Inc()
	metrics.SweepAmountTotal.WithLabelValues(chain).Add(sweepAmount)
	audit.SweepExecuted(chain, fmt.Sprintf("%.8f", sweepAmount), txHash.String(), s.coldAddrBTC)
}

// sweepETH transfers ETH from the hot wallet private key to cold storage.
func (s *Sweeper) sweepETH(ctx context.Context) {
	const chain = "eth"
	start := time.Now()
	defer func() {
		slog.Info("Sweep ETH done", "elapsed", time.Since(start).String())
	}()

	// Parse hot wallet private key.
	keyBytes, err := hex.DecodeString(strings.TrimPrefix(s.ethHotKeyHex, "0x"))
	if err != nil {
		slog.Error("Sweep ETH: invalid hot wallet key hex", "err", err)
		metrics.SweepETHTTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, "0", "invalid hot key: "+err.Error())
		return
	}
	hotKey, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		slog.Error("Sweep ETH: failed to parse hot wallet key", "err", err)
		metrics.SweepETHTTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, "0", "parse key: "+err.Error())
		return
	}

	fromAddr := crypto.PubkeyToAddress(hotKey.PublicKey)

	balance, err := s.ethRPC.Get().BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		slog.Error("Sweep ETH: failed to get balance", "err", err)
		metrics.SweepETHTTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, "0", "balance query: "+err.Error())
		return
	}

	weiPerETH := new(big.Float).SetFloat64(1e18)
	balanceETH, _ := new(big.Float).Quo(new(big.Float).SetInt(balance), weiPerETH).Float64()
	slog.Info("Sweep ETH: current hot wallet balance", "balance", fmt.Sprintf("%.8f", balanceETH))

	// Estimate gas.
	gasPrice, err := s.ethRPC.Get().SuggestGasPrice(ctx)
	if err != nil {
		slog.Error("Sweep ETH: failed to get gas price", "err", err)
		metrics.SweepETHTTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, fmt.Sprintf("%.8f", balanceETH), "gas price: "+err.Error())
		return
	}

	const gasLimit = uint64(21000)
	gasCostWei := new(big.Int).Mul(big.NewInt(int64(gasLimit)), gasPrice)
	gasCostETH, _ := new(big.Float).Quo(new(big.Float).SetInt(gasCostWei), weiPerETH).Float64()

	// Convert keepETH to wei
	keepETHBF := new(big.Float).Mul(new(big.Float).SetFloat64(s.keepETH), new(big.Float).SetFloat64(1e18))
	keepWeiBig, _ := keepETHBF.Int(nil)

	sweepWei := new(big.Int).Sub(balance, gasCostWei)
	sweepWei.Sub(sweepWei, keepWeiBig)

	if sweepWei.Sign() <= 0 {
		slog.Info("Sweep ETH: balance too low after gas + keep",
			"balance", fmt.Sprintf("%.8f", balanceETH),
			"keep", fmt.Sprintf("%.8f", s.keepETH),
			"gas_cost", fmt.Sprintf("%.8f", gasCostETH))
		metrics.SweepETHTTotal.WithLabelValues("skipped").Inc()
		return
	}

	sweepETH, _ := new(big.Float).Quo(new(big.Float).SetInt(sweepWei), weiPerETH).Float64()

	select {
	case <-ctx.Done():
		return
	default:
	}

	// Build and sign transaction.
	nonce, err := s.ethRPC.Get().PendingNonceAt(ctx, fromAddr)
	if err != nil {
		slog.Error("Sweep ETH: failed to get nonce", "err", err)
		metrics.SweepETHTTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, fmt.Sprintf("%.8f", sweepETH), "nonce: "+err.Error())
		return
	}

	chainID, err := s.ethRPC.Get().ChainID(ctx)
	if err != nil {
		slog.Error("Sweep ETH: failed to get chain ID", "err", err)
		metrics.SweepETHTTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, fmt.Sprintf("%.8f", sweepETH), "chainID: "+err.Error())
		return
	}

	coldAddr := common.HexToAddress(s.coldAddrETH)

	tx := types.NewTransaction(nonce, coldAddr, sweepWei, gasLimit, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), hotKey)
	if err != nil {
		slog.Error("Sweep ETH: failed to sign tx", "err", err)
		metrics.SweepETHTTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, fmt.Sprintf("%.8f", sweepETH), "sign: "+err.Error())
		return
	}

	if err := s.ethRPC.Get().SendTransaction(ctx, signedTx); err != nil {
		slog.Error("Sweep ETH: broadcast failed", "err", err)
		metrics.SweepETHTTotal.WithLabelValues("error").Inc()
		audit.SweepFailed(chain, fmt.Sprintf("%.8f", sweepETH), "broadcast: "+err.Error())
		return
	}

	txHash := signedTx.Hash().Hex()

	slog.Warn("Sweep ETH: funds transferred to cold storage",
		"amount", fmt.Sprintf("%.8f", sweepETH),
		"cold_addr", s.coldAddrETH,
		"tx_id", txHash,
	)
	metrics.SweepETHTTotal.WithLabelValues("success").Inc()
	metrics.SweepAmountTotal.WithLabelValues(chain).Add(sweepETH)
	audit.SweepExecuted(chain, fmt.Sprintf("%.8f", sweepETH), txHash, s.coldAddrETH)
}
