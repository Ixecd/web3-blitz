package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Ixecd/web3-blitz/internal/auth"
	"github.com/Ixecd/web3-blitz/internal/code"
	"github.com/Ixecd/web3-blitz/internal/db"
	"github.com/Ixecd/web3-blitz/internal/metrics"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
)

func (h *Handler) GenerateAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		UserID string      `json:"user_id"`
		Chain  types.Chain `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.UserID == "" {
		FailMsg(w, code.ErrInvalidArg, "user_id 不能为空")
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
		Fail(w, code.ErrWalletChainNotSupported)
		return
	}

	if genErr != nil {
		FailMsg(w, code.ErrInternal, genErr.Error())
		return
	}

	OK(w, resp)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}

	address := r.URL.Query().Get("address")
	chainStr := r.URL.Query().Get("chain")

	if address == "" || chainStr == "" {
		FailMsg(w, code.ErrInvalidArg, "缺少 address 或 chain 参数")
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
		Fail(w, code.ErrWalletChainNotSupported)
		return
	}

	if err != nil {
		FailMsg(w, code.ErrInternal, err.Error())
		return
	}

	OK(w, resp)
}

func (h *Handler) ListDeposits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		FailMsg(w, code.ErrInvalidArg, "缺少 user_id 参数")
		return
	}

	deposits, err := h.queries.ListDepositsByUserID(r.Context(), userID)
	if err != nil {
		FailMsg(w, code.ErrInternal, err.Error())
		return
	}

	OK(w, deposits)
}

func (h *Handler) GetTotalBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}

	userID := r.URL.Query().Get("user_id")
	chainStr := r.URL.Query().Get("chain")

	if userID == "" || chainStr == "" {
		FailMsg(w, code.ErrInvalidArg, "缺少 user_id 或 chain 参数")
		return
	}

	total, err := h.queries.GetTotalDepositByUserIDAndChain(r.Context(), db.GetTotalDepositByUserIDAndChainParams{
		UserID: userID,
		Chain:  chainStr,
	})
	if err != nil {
		FailMsg(w, code.ErrInternal, err.Error())
		return
	}

	var totalFloat float64
	if t, ok := total.(string); ok {
		fmt.Sscanf(t, "%f", &totalFloat)
	} else {
		totalFloat, _ = total.(float64)
	}

	OK(w, map[string]interface{}{
		"user_id": userID,
		"chain":   chainStr,
		"total":   totalFloat,
	})
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		UserID    string      `json:"user_id"`
		ToAddress string      `json:"to_address"`
		Amount    float64     `json:"amount"`
		Chain     types.Chain `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.UserID == "" || req.ToAddress == "" || req.Amount <= 0 {
		FailMsg(w, code.ErrInvalidArg, "user_id / to_address / amount 不能为空或非正数")
		return
	}

	ctx := r.Context()

	// 分布式锁，防止重复提币
	lockKey := fmt.Sprintf("withdraw:%s:%s", req.UserID, req.Chain)
	l, err := h.locker.Acquire(ctx, lockKey)
	if err != nil {
		metrics.LockAcquireFailTotal.WithLabelValues(lockKey).Inc()
		Fail(w, code.ErrWalletDuplicateWithdraw)
		return
	}
	defer l.Release(context.Background())

	// 类型转换
	toFloat := func(v interface{}) float64 {
		switch val := v.(type) {
		case float64:
			return val
		case int64:
			return float64(val)
		case string:
			var f float64
			fmt.Sscanf(val, "%f", &f)
			return f
		case []byte:
			var f float64
			fmt.Sscanf(string(val), "%f", &f)
			return f
		}
		return 0
	}

	// 余额校验
	rawDeposit, err := h.queries.GetTotalDepositByUserIDAndChain(ctx, db.GetTotalDepositByUserIDAndChainParams{
		UserID: req.UserID,
		Chain:  string(req.Chain),
	})
	if err != nil {
		FailMsg(w, code.ErrInternal, "查询充值余额失败: "+err.Error())
		return
	}

	rawWithdrawal, err := h.queries.GetTotalWithdrawalByUserIDAndChain(ctx, db.GetTotalWithdrawalByUserIDAndChainParams{
		UserID: req.UserID,
		Chain:  string(req.Chain),
	})
	if err != nil {
		FailMsg(w, code.ErrInternal, "查询提币余额失败: "+err.Error())
		return
	}

	available := toFloat(rawDeposit) - toFloat(rawWithdrawal)
	if available < req.Amount {
		FailMsg(w, code.ErrWalletInsufficientBalance,
			fmt.Sprintf("余额不足: 可用 %.8f，请求 %.8f", available, req.Amount))
		return
	}

	// 限额校验
	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}

	userLevel, err := h.queries.GetUserLevel(ctx, claims.UserID)
	if err != nil {
		FailMsg(w, code.ErrInternal, "查询用户等级失败: "+err.Error())
		return
	}

	limit, err := h.queries.GetWithdrawalLimit(ctx, int32(userLevel))
	if err != nil {
		FailMsg(w, code.ErrInternal, "查询提币限额失败: "+err.Error())
		return
	}

	rawUsed, err := h.queries.GetLast24hWithdrawalByUserAndChain(ctx, db.GetLast24hWithdrawalByUserAndChainParams{
		UserID: req.UserID,
		Chain:  string(req.Chain),
	})
	if err != nil {
		FailMsg(w, code.ErrInternal, "查询提币额度失败: "+err.Error())
		return
	}

	usedToday := toFloat(rawUsed)
	var dailyLimit float64
	switch req.Chain {
	case types.ChainBTC:
		fmt.Sscanf(limit.BtcDaily, "%f", &dailyLimit)
	case types.ChainETH:
		fmt.Sscanf(limit.EthDaily, "%f", &dailyLimit)
	}

	if usedToday+req.Amount > dailyLimit {
		FailMsg(w, code.ErrWalletDailyLimitExceeded,
			fmt.Sprintf("超出每日提币限额: 已用 %.8f，本次 %.8f，限额 %.8f（%s）",
				usedToday, req.Amount, dailyLimit, limit.LevelName))
		return
	}

	// 写入 pending 记录
	record, err := h.queries.CreateWithdrawal(ctx, db.CreateWithdrawalParams{
		UserID:  req.UserID,
		Address: req.ToAddress,
		Amount:  fmt.Sprintf("%.8f", req.Amount),
		Chain:   string(req.Chain),
	})
	if err != nil {
		FailMsg(w, code.ErrInternal, "创建提币记录失败: "+err.Error())
		return
	}

	// 广播交易
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
		Fail(w, code.ErrWalletChainNotSupported)
		return
	}

	// 更新 DB 状态
	status := "completed"
	if broadcastErr != nil {
		status = "failed"
		slog.Error("提币广播失败", "id", record.ID, "err", broadcastErr)
	}

	_ = h.queries.UpdateWithdrawalTx(ctx, db.UpdateWithdrawalTxParams{
		TxID:   sql.NullString{String: txID, Valid: txID != ""},
		Fee:    fmt.Sprintf("%.8f", fee),
		Status: status,
		ID:     record.ID,
	})

	if broadcastErr != nil {
		metrics.WithdrawTotal.WithLabelValues(string(req.Chain), "failed").Inc()
	} else {
		metrics.WithdrawTotal.WithLabelValues(string(req.Chain), "completed").Inc()
		metrics.WithdrawAmount.WithLabelValues(string(req.Chain)).Add(req.Amount)
	}

	OK(w, map[string]interface{}{
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

func (h *Handler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		FailMsg(w, code.ErrInvalidArg, "缺少 user_id 参数")
		return
	}

	withdrawals, err := h.queries.ListWithdrawalsByUserID(r.Context(), userID)
	if err != nil {
		FailMsg(w, code.ErrInternal, err.Error())
		return
	}

	type WithdrawalResp struct {
		ID        int64   `json:"id"`
		TxID      string  `json:"tx_id"`
		Address   string  `json:"address"`
		UserID    string  `json:"user_id"`
		Amount    float64 `json:"amount"`
		Fee       float64 `json:"fee"`
		Status    string  `json:"status"`
		Chain     string  `json:"chain"`
		CreatedAt string  `json:"created_at"`
	}

	resp := make([]WithdrawalResp, 0, len(withdrawals))
	for _, wl := range withdrawals {
		var amount, fee float64
		fmt.Sscanf(wl.Amount, "%f", &amount)
		fmt.Sscanf(wl.Fee, "%f", &fee)
		resp = append(resp, WithdrawalResp{
			ID:        wl.ID,
			TxID:      wl.TxID.String,
			Address:   wl.Address,
			UserID:    wl.UserID,
			Amount:    amount,
			Fee:       fee,
			Status:    wl.Status,
			Chain:     wl.Chain,
			CreatedAt: wl.CreatedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	OK(w, resp)
}
