package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Ixecd/blitz/internal/audit"
	"github.com/Ixecd/blitz/internal/auth"
	"github.com/Ixecd/blitz/internal/code"
	"github.com/Ixecd/blitz/internal/db"
	"github.com/Ixecd/blitz/internal/metrics"
	"github.com/Ixecd/blitz/internal/wallet/types"
	"github.com/ethereum/go-ethereum/common"
)

func (h *Handler) GenerateAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}

	var req struct {
		Chain types.Chain `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}

	userID := fmt.Sprintf("%d", claims.UserID)

	var resp types.AddressResponse
	var genErr error

	switch req.Chain {
	case types.ChainBTC:
		resp, genErr = h.btcWallet.GenerateDepositAddress(r.Context(), userID, req.Chain)
	case types.ChainETH:
		resp, genErr = h.ethWallet.GenerateDepositAddress(r.Context(), userID, req.Chain)
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
		FailInternal(w, err)
		return
	}

	OK(w, resp)
}

func (h *Handler) ListDeposits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}

	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}
	userID := fmt.Sprintf("%d", claims.UserID)

	deposits, err := h.queries.ListDepositsByUserID(r.Context(), userID)
	if err != nil {
		FailInternal(w, err)
		return
	}

	OK(w, deposits)
}

func (h *Handler) GetTotalBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}

	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}

	chainStr := r.URL.Query().Get("chain")
	if chainStr == "" {
		FailMsg(w, code.ErrInvalidArg, "缺少 chain 参数")
		return
	}

	userID := fmt.Sprintf("%d", claims.UserID)

	total, err := h.queries.GetTotalDepositByUserIDAndChain(r.Context(), db.GetTotalDepositByUserIDAndChainParams{
		UserID: userID,
		Chain:  chainStr,
	})
	if err != nil {
		FailInternal(w, err)
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
		ToAddress string      `json:"to_address"`
		Amount    float64     `json:"amount"`
		Chain     types.Chain `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.ToAddress == "" || req.Amount <= 0 {
		FailMsg(w, code.ErrInvalidArg, "to_address / amount 不能为空或非正数")
		return
	}
	// 目标地址合法性校验
	if req.Chain == types.ChainETH && !common.IsHexAddress(req.ToAddress) {
		FailMsg(w, code.ErrInvalidArg, "ETH 地址格式不合法")
		return
	}

	ctx := r.Context()
	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}
	userID := fmt.Sprintf("%d", claims.UserID)

	// 单笔提币上限（BTC）
	const maxSingleBTC = 1.0
	const maxSingleETH = 10.0
	switch req.Chain {
	case types.ChainBTC:
		if req.Amount > maxSingleBTC {
			FailMsg(w, code.ErrWalletInsufficientBalance,
				fmt.Sprintf("单笔提币上限 %.1f BTC，本次 %.8f", maxSingleBTC, req.Amount))
			return
		}
	case types.ChainETH:
		if req.Amount > maxSingleETH {
			FailMsg(w, code.ErrWalletInsufficientBalance,
				fmt.Sprintf("单笔提币上限 %.1f ETH，本次 %.8f", maxSingleETH, req.Amount))
			return
		}
	}

	// 分布式锁，防止重复提币
	lockKey := fmt.Sprintf("withdraw:%s:%s", userID, req.Chain)
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
		UserID: userID,
		Chain:  string(req.Chain),
	})
	if err != nil {
		FailInternal(w, err)
		return
	}

	rawWithdrawal, err := h.queries.GetTotalWithdrawalByUserIDAndChain(ctx, db.GetTotalWithdrawalByUserIDAndChainParams{
		UserID: userID,
		Chain:  string(req.Chain),
	})
	if err != nil {
		FailInternal(w, err)
		return
	}

	available := toFloat(rawDeposit) - toFloat(rawWithdrawal)
	if available < req.Amount {
		FailMsg(w, code.ErrWalletInsufficientBalance,
			fmt.Sprintf("余额不足: 可用 %.8f，请求 %.8f", available, req.Amount))
		return
	}

	// 限额校验
	userLevel, err := h.queries.GetUserLevel(ctx, claims.UserID)
	if err != nil {
		FailInternal(w, err)
		return
	}

	limit, err := h.queries.GetWithdrawalLimit(ctx, int32(userLevel))
	if err != nil {
		FailInternal(w, err)
		return
	}

	rawUsed, err := h.queries.GetLast24hWithdrawalByUserAndChain(ctx, db.GetLast24hWithdrawalByUserAndChainParams{
		UserID: userID,
		Chain:  string(req.Chain),
	})
	if err != nil {
		FailInternal(w, err)
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
		UserID:  userID,
		Address: req.ToAddress,
		Amount:  fmt.Sprintf("%.8f", req.Amount),
		Chain:   string(req.Chain),
	})
	if err != nil {
		FailInternal(w, err)
		return
	}

	audit.WithdrawSubmitted(userID, string(req.Chain),
		fmt.Sprintf("%.8f", req.Amount), req.ToAddress)

	// 是否自动广播（WITHDRAWAL_AUTO_APPROVE=true 时保持老行为）
	if os.Getenv("WITHDRAWAL_AUTO_APPROVE") == "true" {
		txID, fee, broadcastErr := h.broadcastWithdrawal(ctx, record.ID, req.Chain, req.ToAddress, req.Amount)
		status := "completed"
		if broadcastErr != nil {
			status = "failed"
			slog.Error("提币广播失败", "id", record.ID, "err", broadcastErr)
			metrics.WithdrawTotal.WithLabelValues(string(req.Chain), "failed").Inc()
			audit.WithdrawFailed(userID, string(req.Chain),
				fmt.Sprintf("%.8f", req.Amount), broadcastErr.Error())
		} else {
			metrics.WithdrawTotal.WithLabelValues(string(req.Chain), "completed").Inc()
			metrics.WithdrawAmount.WithLabelValues(string(req.Chain)).Add(req.Amount)
			audit.WithdrawCompleted(userID, string(req.Chain),
				fmt.Sprintf("%.8f", req.Amount), txID)
		}
		OK(w, map[string]interface{}{
			"id":         record.ID,
			"tx_id":      txID,
			"user_id":    userID,
			"to_address": req.ToAddress,
			"amount":     req.Amount,
			"fee":        fee,
			"status":     status,
			"chain":      req.Chain,
		})
		return
	}

	// 人工审核模式：只写 pending，不广播
	slog.Info("提币已提交，等待管理员审核", "id", record.ID, "user_id", userID,
		"chain", req.Chain, "amount", fmt.Sprintf("%.8f", req.Amount))
	OK(w, map[string]interface{}{
		"id":         record.ID,
		"user_id":    userID,
		"to_address": req.ToAddress,
		"amount":     req.Amount,
		"status":     "pending_review",
		"chain":      req.Chain,
		"note":       "提币请求已提交，等待管理员审核。",
	})
}

func (h *Handler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}

	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}
	userID := fmt.Sprintf("%d", claims.UserID)

	withdrawals, err := h.queries.ListWithdrawalsByUserID(r.Context(), userID)
	if err != nil {
		FailInternal(w, err)
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

// broadcastWithdrawal 执行链上广播并更新 DB 状态。
// 供 auto-approve 模式和人工审核通过后复用。
func (h *Handler) broadcastWithdrawal(ctx context.Context, withdrawalID int64, chain types.Chain, toAddress string, amount float64) (txID string, fee float64, err error) {
	switch chain {
	case types.ChainBTC:
		res, e := h.btcWallet.Withdraw(ctx, toAddress, amount)
		if e != nil {
			_ = h.queries.AdminApproveRejectWithdrawal(ctx, db.AdminApproveRejectWithdrawalParams{
				TxID:   sql.NullString{},
				Fee:    "0.00000000",
				Status: "failed",
				ID:     withdrawalID,
			})
			return res.TxID, res.Fee, e
		}
		txID, fee = res.TxID, res.Fee
	case types.ChainETH:
		res, e := h.ethWallet.Withdraw(ctx, toAddress, amount)
		if e != nil {
			_ = h.queries.AdminApproveRejectWithdrawal(ctx, db.AdminApproveRejectWithdrawalParams{
				TxID:   sql.NullString{},
				Fee:    "0.00000000",
				Status: "failed",
				ID:     withdrawalID,
			})
			return res.TxID, res.Fee, e
		}
		txID, fee = res.TxID, res.Fee
	default:
		return "", 0, fmt.Errorf("unsupported chain: %s", chain)
	}

	return txID, fee, nil
}

// ListPendingWithdrawals 管理员查看所有待审核提币（需要 withdraw:review 权限）
func (h *Handler) ListPendingWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}

	withdrawals, err := h.queries.ListPendingWithdrawals(r.Context())
	if err != nil {
		FailInternal(w, err)
		return
	}

	type PendingResp struct {
		ID        int64  `json:"id"`
		Address   string `json:"address"`
		UserID    string `json:"user_id"`
		Amount    string `json:"amount"`
		Status    string `json:"status"`
		Chain     string `json:"chain"`
		CreatedAt string `json:"created_at"`
	}

	resp := make([]PendingResp, 0, len(withdrawals))
	for _, wl := range withdrawals {
		resp = append(resp, PendingResp{
			ID:        wl.ID,
			Address:   wl.Address,
			UserID:    wl.UserID,
			Amount:    wl.Amount,
			Status:    wl.Status,
			Chain:     wl.Chain,
			CreatedAt: wl.CreatedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	OK(w, resp)
}

// ApproveWithdrawal 管理员审核通过提币，执行链上广播（需要 withdraw:review 权限）
func (h *Handler) ApproveWithdrawal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.ID <= 0 {
		FailMsg(w, code.ErrInvalidArg, "id 必须为正整数")
		return
	}

	ctx := r.Context()
	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}
	adminID := fmt.Sprintf("%d", claims.UserID)

	// 从 DB 加载 pending 记录确认状态
	pendings, err := h.queries.ListPendingWithdrawals(ctx)
	if err != nil {
		FailInternal(w, err)
		return
	}

	var wl db.Withdrawal
	found := false
	for _, p := range pendings {
		if p.ID == req.ID {
			wl = p
			found = true
			break
		}
	}
	if !found {
		FailMsg(w, code.ErrWalletInvalidStatus, "该提币记录不存在或已处理")
		return
	}

	var amountFloat float64
	fmt.Sscanf(wl.Amount, "%f", &amountFloat)

	chain := types.Chain(wl.Chain)
	txID, fee, broadcastErr := h.broadcastWithdrawal(ctx, wl.ID, chain, wl.Address, amountFloat)

	if broadcastErr != nil {
		slog.Error("审核通过后广播失败", "id", wl.ID, "err", broadcastErr)
		metrics.WithdrawTotal.WithLabelValues(wl.Chain, "failed").Inc()
		audit.WithdrawFailed(wl.UserID, wl.Chain, wl.Amount, broadcastErr.Error())
		Fail(w, code.ErrWalletBroadcastFailed)
		return
	}

	_ = h.queries.AdminApproveRejectWithdrawal(ctx, db.AdminApproveRejectWithdrawalParams{
		TxID:   sql.NullString{String: txID, Valid: true},
		Fee:    fmt.Sprintf("%.8f", fee),
		Status: "completed",
		ID:     wl.ID,
	})

	metrics.WithdrawTotal.WithLabelValues(wl.Chain, "completed").Inc()
	metrics.WithdrawAmount.WithLabelValues(wl.Chain).Add(amountFloat)
	audit.WithdrawApproved(wl.UserID, wl.Chain, wl.Amount, txID, adminID)

	slog.Info("提币审核通过并已广播", "id", wl.ID, "admin", adminID, "tx_id", txID,
		"chain", wl.Chain, "amount", wl.Amount)

	OK(w, map[string]interface{}{
		"id":      wl.ID,
		"tx_id":   txID,
		"status":  "completed",
		"message": "提币审核通过，已广播。",
	})
}

// RejectWithdrawal 管理员拒绝提币（需要 withdraw:review 权限）
func (h *Handler) RejectWithdrawal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		ID     int64  `json:"id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.ID <= 0 {
		FailMsg(w, code.ErrInvalidArg, "id 必须为正整数")
		return
	}
	if req.Reason == "" {
		req.Reason = "管理员拒绝"
	}

	ctx := r.Context()
	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}
	adminID := fmt.Sprintf("%d", claims.UserID)

	pendings, err := h.queries.ListPendingWithdrawals(ctx)
	if err != nil {
		FailInternal(w, err)
		return
	}

	var wl db.Withdrawal
	found := false
	for _, p := range pendings {
		if p.ID == req.ID {
			wl = p
			found = true
			break
		}
	}
	if !found {
		FailMsg(w, code.ErrWalletInvalidStatus, "该提币记录不存在或已处理")
		return
	}

	_ = h.queries.AdminApproveRejectWithdrawal(ctx, db.AdminApproveRejectWithdrawalParams{
		TxID:   sql.NullString{},
		Fee:    "0.00000000",
		Status: "rejected",
		ID:     wl.ID,
	})

	metrics.WithdrawTotal.WithLabelValues(wl.Chain, "rejected").Inc()
	audit.WithdrawRejected(wl.UserID, wl.Chain, wl.Amount, adminID, req.Reason)

	slog.Warn("提币已被管理员拒绝", "id", wl.ID, "admin", adminID,
		"chain", wl.Chain, "amount", wl.Amount, "reason", req.Reason)

	OK(w, map[string]interface{}{
		"id":      wl.ID,
		"status":  "rejected",
		"reason":  req.Reason,
		"message": "提币已拒绝。",
	})
}
