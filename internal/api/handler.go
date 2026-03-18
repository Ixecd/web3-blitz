package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/Ixecd/web3-blitz/internal/db"
	"github.com/Ixecd/web3-blitz/internal/wallet/btc"
	"github.com/Ixecd/web3-blitz/internal/wallet/eth"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
)

type Handler struct {
	btcWallet *btc.BTCWallet
	ethWallet *eth.ETHWallet
	queries   *db.Queries
}

func NewHandler(btcWallet *btc.BTCWallet, ethWallet *eth.ETHWallet, queries *db.Queries) *Handler {
	return &Handler{
		btcWallet: btcWallet,
		ethWallet: ethWallet,
		queries:   queries,
	}
}

func (h *Handler) GenerateAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string      `json:"user_id"`
		Chain  types.Chain `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "参数错误", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, "user_id 不能为空", http.StatusBadRequest)
		return
	}

	var resp types.AddressResponse
	var genErr error

	switch req.Chain {
	case types.ChainBTC:
		resp, genErr = h.btcWallet.GenerateDepositAddress(r.Context(), req.UserID, req.Chain)
	case types.ChainETH:
		resp, genErr = h.ethWallet.GenerateDepositAddress(r.Context(), req.UserID, req.Chain)
	default:
		http.Error(w, "不支持的链", http.StatusBadRequest)
		return
	}

	if genErr != nil {
		http.Error(w, genErr.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "只支持 GET", http.StatusMethodNotAllowed)
		return
	}

	address := r.URL.Query().Get("address")
	chainStr := r.URL.Query().Get("chain")

	if address == "" || chainStr == "" {
		http.Error(w, "缺少 address 或 chain 参数", http.StatusBadRequest)
		return
	}

	var resp types.BalanceResponse
	var err error

	switch chainStr {
	case "btc":
		resp, err = h.btcWallet.GetBalance(r.Context(), address, types.ChainBTC)
	case "eth":
		resp, err = h.ethWallet.GetBalance(r.Context(), address, types.ChainETH)
	default:
		http.Error(w, "不支持的链", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) ListDeposits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "只支持 GET", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "缺少 user_id 参数", http.StatusBadRequest)
		return
	}

	deposits, err := h.queries.ListDepositsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(deposits)
}

func (h *Handler) GetTotalBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "只支持 GET", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	chainStr := r.URL.Query().Get("chain")

	if userID == "" || chainStr == "" {
		http.Error(w, "缺少 user_id 或 chain 参数", http.StatusBadRequest)
		return
	}

	total, err := h.queries.GetTotalDepositByUserIDAndChain(r.Context(), db.GetTotalDepositByUserIDAndChainParams{
		UserID: userID,
		Chain:  chainStr,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"chain":   chainStr,
		"total":   total,
	})
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID    string      `json:"user_id"`
		ToAddress string      `json:"to_address"`
		Amount    float64     `json:"amount"`
		Chain     types.Chain `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "参数错误", http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.ToAddress == "" || req.Amount <= 0 {
		http.Error(w, "user_id / to_address / amount 不能为空或非正数", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// 1. 写入 pending 记录，拿到 ID
	record, err := h.queries.CreateWithdrawal(ctx, db.CreateWithdrawalParams{
		UserID:  req.UserID,
		Address: req.ToAddress,
		Amount:  req.Amount,
		Chain:   string(req.Chain),
	})
	if err != nil {
		http.Error(w, "创建提币记录失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. 广播交易
	var txID string
	var fee float64
	var broadcastErr error

	switch req.Chain {
	case types.ChainBTC:
		res, err := h.btcWallet.Withdraw(ctx, req.ToAddress, req.Amount)
		txID, fee, broadcastErr = res.TxID, res.Fee, err
	case types.ChainETH:
		res, err := h.ethWallet.Withdraw(ctx, req.ToAddress, req.Amount)
		txID, fee, broadcastErr = res.TxID, res.Fee, err
	default:
		http.Error(w, "不支持的链", http.StatusBadRequest)
		return
	}

	// 3. 更新 DB 状态
	status := "completed"
	if broadcastErr != nil {
		status = "failed"
		log.Printf("[ERROR] 提币广播失败 id=%d: %v", record.ID, broadcastErr)
	}

	_ = h.queries.UpdateWithdrawalTx(ctx, db.UpdateWithdrawalTxParams{
		TxID:   sql.NullString{String: txID, Valid: txID != ""},
		Fee:    fee,
		Status: status,
		ID:     record.ID,
	})

	if broadcastErr != nil {
		http.Error(w, "提币失败: "+broadcastErr.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         record.ID,
		"tx_id":      txID,
		"user_id":    req.UserID,
		"to_address": req.ToAddress,
		"amount":     req.Amount,
		"fee":        fee,
		"status":     status,
		"chain":      req.Chain,
	})
}
