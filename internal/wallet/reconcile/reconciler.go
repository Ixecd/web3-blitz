package reconcile

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/Ixecd/blitz/internal/config"
	"github.com/Ixecd/blitz/internal/db"
	"github.com/Ixecd/blitz/internal/metrics"
	"github.com/Ixecd/blitz/internal/wallet/btc"
	btcutil "github.com/btcsuite/btcd/btcutil"
	"github.com/ethereum/go-ethereum/common"
)

// Reconciler performs on-chain vs DB reconciliation for BTC and ETH deposits.
type Reconciler struct {
	queries  *db.Queries
	btcRPC   *config.BTCRPCHolder
	ethRPC   *config.ETHRPCHolder
	btcDepth int64
	ethDepth uint64
}

// New creates a Reconciler with default scan depths (BTC: 200 blocks, ETH: 10000 blocks).
func New(queries *db.Queries, btcRPC *config.BTCRPCHolder, ethRPC *config.ETHRPCHolder) *Reconciler {
	r := &Reconciler{
		queries:  queries,
		btcRPC:   btcRPC,
		ethRPC:   ethRPC,
		btcDepth: 200,
		ethDepth: 10000,
	}

	if v, err := strconv.ParseInt(os.Getenv("RECONCILE_BLOCK_DEPTH_BTC"), 10, 64); err == nil && v > 0 {
		r.btcDepth = v
	}
	if v, err := strconv.ParseUint(os.Getenv("RECONCILE_BLOCK_DEPTH_ETH"), 10, 64); err == nil && v > 0 {
		r.ethDepth = v
	}

	return r
}

// SetBTCDepth overrides the default BTC scan depth.
func (r *Reconciler) SetBTCDepth(d int64) { r.btcDepth = d }

// SetETHDepth overrides the default ETH scan depth.
func (r *Reconciler) SetETHDepth(d uint64) { r.ethDepth = d }

// Run loads all deposit addresses from DB and runs reconciliation for each chain.
func (r *Reconciler) Run(ctx context.Context) {
	addrs, err := r.queries.ListAllDepositAddresses(ctx)
	if err != nil {
		slog.Error("Reconcile: failed to load deposit addresses", "err", err)
		metrics.ReconcileErrorsTotal.WithLabelValues("all", "load_addresses").Inc()
		return
	}

	var btcAddrs, ethAddrs []string
	for _, a := range addrs {
		switch a.Chain {
		case "btc":
			btcAddrs = append(btcAddrs, a.Address)
		case "eth":
			ethAddrs = append(ethAddrs, a.Address)
		}
	}

	slog.Info("Reconciliation started", "btc_addrs", len(btcAddrs), "eth_addrs", len(ethAddrs),
		"btc_depth", r.btcDepth, "eth_depth", r.ethDepth)

	if len(btcAddrs) > 0 {
		r.reconcileBTC(ctx, btcAddrs)
	}
	if len(ethAddrs) > 0 {
		r.reconcileETH(ctx, ethAddrs)
	}

	slog.Info("Reconciliation finished")
}

// reconcileBTC scans recent BTC blocks and compares chain transactions against DB records.
func (r *Reconciler) reconcileBTC(ctx context.Context, addresses []string) {
	const chain = "btc"
	start := time.Now()
	defer func() {
		metrics.ReconcileDuration.WithLabelValues(chain).Set(time.Since(start).Seconds())
	}()

	addrSet := make(map[string]bool, len(addresses))
	for _, a := range addresses {
		addrSet[a] = true
	}

	info, err := r.btcRPC.Get().GetBlockChainInfo()
	if err != nil {
		slog.Error("Reconcile BTC: failed to get blockchain info", "err", err)
		metrics.ReconcileErrorsTotal.WithLabelValues(chain, "get_info").Inc()
		return
	}
	currentHeight := int64(info.Blocks)
	minHeight := currentHeight - r.btcDepth
	if minHeight < 0 {
		minHeight = 0
	}

	// Load DB deposits within the scan range.
	dbDeposits, err := r.queries.ListDepositsByChainAndHeightRange(ctx, db.ListDepositsByChainAndHeightRangeParams{
		Chain:     chain,
		MinHeight: minHeight,
		MaxHeight: currentHeight,
	})
	if err != nil {
		slog.Error("Reconcile BTC: failed to load DB deposits", "err", err)
		metrics.ReconcileErrorsTotal.WithLabelValues(chain, "load_db").Inc()
		return
	}

	dbByTxID := make(map[string]db.Deposit, len(dbDeposits))
	for _, d := range dbDeposits {
		dbByTxID[d.TxID] = d
	}

	// Track which DB tx_ids were seen on chain.
	seenOnChain := make(map[string]bool)

	netParams := btc.NetParams()

	for h := minHeight; h <= currentHeight; h++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		hash, err := r.btcRPC.Get().GetBlockHash(h)
		if err != nil {
			slog.Warn("Reconcile BTC: failed to get block hash", "height", h, "err", err)
			metrics.ReconcileErrorsTotal.WithLabelValues(chain, "get_block_hash").Inc()
			continue
		}

		block, err := r.btcRPC.Get().GetBlockVerboseTx(hash)
		if err != nil {
			slog.Warn("Reconcile BTC: failed to get block", "height", h, "err", err)
			metrics.ReconcileErrorsTotal.WithLabelValues(chain, "get_block").Inc()
			continue
		}

		for _, tx := range block.Tx {
			for _, vout := range tx.Vout {
				if len(vout.ScriptPubKey.Address) == 0 {
					continue
				}
				address := vout.ScriptPubKey.Address

				if !addrSet[address] {
					continue
				}

				if _, err := btcutil.DecodeAddress(address, netParams); err != nil {
					continue
				}

				chainAmount := fmt.Sprintf("%.8f", vout.Value)

				dbDep, inDB := dbByTxID[tx.Txid]
				if !inDB {
					// Missing: on chain but not in DB.
					metrics.ReconcileMissingDeposits.WithLabelValues(chain).Inc()
					slog.Warn("Reconcile finding",
						"type", "missing",
						"chain", chain,
						"tx_id", tx.Txid,
						"address", address,
						"amount", chainAmount,
						"height", h,
					)
				} else {
					seenOnChain[tx.Txid] = true
					if dbDep.Amount != chainAmount {
						metrics.ReconcileAmountMismatch.WithLabelValues(chain).Inc()
						slog.Warn("Reconcile finding",
							"type", "amount_mismatch",
							"chain", chain,
							"tx_id", tx.Txid,
							"address", address,
							"db_amount", dbDep.Amount,
							"chain_amount", chainAmount,
							"height", h,
						)
					}
				}
			}
		}
	}

	// Phantom: in DB but not seen on chain within the scan range.
	for txID, dep := range dbByTxID {
		if !seenOnChain[txID] {
			metrics.ReconcilePhantomDeposits.WithLabelValues(chain).Inc()
			slog.Warn("Reconcile finding",
				"type", "phantom",
				"chain", chain,
				"tx_id", dep.TxID,
				"address", dep.Address,
				"amount", dep.Amount,
				"height", dep.Height,
			)
		}
	}
}

// reconcileETH scans recent ETH blocks and compares chain transactions against DB records.
func (r *Reconciler) reconcileETH(ctx context.Context, addresses []string) {
	const chain = "eth"
	start := time.Now()
	defer func() {
		metrics.ReconcileDuration.WithLabelValues(chain).Set(time.Since(start).Seconds())
	}()

	addrSet := make(map[common.Address]bool, len(addresses))
	for _, a := range addresses {
		if common.IsHexAddress(a) {
			addrSet[common.HexToAddress(a)] = true
		}
	}
	if len(addrSet) == 0 {
		return
	}

	header, err := r.ethRPC.Get().HeaderByNumber(ctx, nil)
	if err != nil {
		slog.Error("Reconcile ETH: failed to get header", "err", err)
		metrics.ReconcileErrorsTotal.WithLabelValues(chain, "get_header").Inc()
		return
	}
	currentHeight := header.Number.Uint64()
	var minHeight uint64
	if currentHeight > r.ethDepth {
		minHeight = currentHeight - r.ethDepth
	}

	// Load DB deposits within the scan range.
	dbDeposits, err := r.queries.ListDepositsByChainAndHeightRange(ctx, db.ListDepositsByChainAndHeightRangeParams{
		Chain:     chain,
		MinHeight: int64(minHeight),
		MaxHeight: int64(currentHeight),
	})
	if err != nil {
		slog.Error("Reconcile ETH: failed to load DB deposits", "err", err)
		metrics.ReconcileErrorsTotal.WithLabelValues(chain, "load_db").Inc()
		return
	}

	dbByTxID := make(map[string]db.Deposit, len(dbDeposits))
	for _, d := range dbDeposits {
		dbByTxID[d.TxID] = d
	}

	seenOnChain := make(map[string]bool)

	weiPerETH := new(big.Float).SetFloat64(1e18)

	for h := minHeight; h <= currentHeight; h++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		block, err := r.ethRPC.Get().BlockByNumber(ctx, new(big.Int).SetUint64(h))
		if err != nil {
			slog.Warn("Reconcile ETH: failed to get block", "height", h, "err", err)
			metrics.ReconcileErrorsTotal.WithLabelValues(chain, "get_block").Inc()
			continue
		}

		for _, tx := range block.Transactions() {
			if tx.To() == nil {
				continue
			}

			toAddr := *tx.To()
			if !addrSet[toAddr] {
				continue
			}

			amountBF := new(big.Float).Quo(new(big.Float).SetInt(tx.Value()), weiPerETH)
			chainAmount := fmt.Sprintf("%.8f", amountBF)

			txHash := tx.Hash().Hex()

			dbDep, inDB := dbByTxID[txHash]
			if !inDB {
				metrics.ReconcileMissingDeposits.WithLabelValues(chain).Inc()
				slog.Warn("Reconcile finding",
					"type", "missing",
					"chain", chain,
					"tx_id", txHash,
					"address", toAddr.Hex(),
					"amount", chainAmount,
					"height", h,
				)
			} else {
				seenOnChain[txHash] = true
				if dbDep.Amount != chainAmount {
					metrics.ReconcileAmountMismatch.WithLabelValues(chain).Inc()
					slog.Warn("Reconcile finding",
						"type", "amount_mismatch",
						"chain", chain,
						"tx_id", txHash,
						"address", toAddr.Hex(),
						"db_amount", dbDep.Amount,
						"chain_amount", chainAmount,
						"height", h,
					)
				}
			}
		}
	}

	// Phantom: in DB but not seen on chain within the scan range.
	for txID, dep := range dbByTxID {
		if !seenOnChain[txID] {
			metrics.ReconcilePhantomDeposits.WithLabelValues(chain).Inc()
			slog.Warn("Reconcile finding",
				"type", "phantom",
				"chain", chain,
				"tx_id", dep.TxID,
				"address", dep.Address,
				"amount", dep.Amount,
				"height", dep.Height,
			)
		}
	}
}
