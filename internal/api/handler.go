package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Ixecd/web3-blitz/internal/auth"
	"github.com/Ixecd/web3-blitz/internal/db"
	"github.com/Ixecd/web3-blitz/internal/lock"
	"github.com/Ixecd/web3-blitz/internal/pkg/code"
	"github.com/Ixecd/web3-blitz/internal/wallet/btc"
	"github.com/Ixecd/web3-blitz/internal/wallet/eth"
	"github.com/Ixecd/web3-blitz/internal/wallet/types"
)

type Handler struct {
	btcWallet *btc.BTCWallet
	ethWallet *eth.ETHWallet
	queries   *db.Queries
	locker    *lock.DistributedLock
	jwtSecret string
}

func NewHandler(btcWallet *btc.BTCWallet, ethWallet *eth.ETHWallet, queries *db.Queries, locker *lock.DistributedLock, jwtSecret string) *Handler {
	return &Handler{
		btcWallet: btcWallet,
		ethWallet: ethWallet,
		queries:   queries,
		locker:    locker,
		jwtSecret: jwtSecret,
	}
}

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
		log.Printf("[ERROR] 提币广播失败 id=%d: %v", record.ID, broadcastErr)
	}

	_ = h.queries.UpdateWithdrawalTx(ctx, db.UpdateWithdrawalTxParams{
		TxID:   sql.NullString{String: txID, Valid: txID != ""},
		Fee:    fmt.Sprintf("%.8f", fee),
		Status: status,
		ID:     record.ID,
	})

	if broadcastErr != nil {
		Fail(w, code.ErrWalletBroadcastFailed)
		return
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

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.Username == "" || req.Password == "" {
		FailMsg(w, code.ErrInvalidArg, "username 和 password 不能为空")
		return
	}
	if len(req.Password) < 8 {
		FailMsg(w, code.ErrInvalidArg, "密码长度不能少于8位")
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Username: req.Username,
		Password: hashed,
	})
	if err != nil {
		Fail(w, code.ErrUserAlreadyExists)
		return
	}

	OK(w, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}

	user, err := h.queries.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		Fail(w, code.ErrUserPasswordWrong)
		return
	}

	if !auth.CheckPassword(req.Password, user.Password) {
		Fail(w, code.ErrUserPasswordWrong)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, h.jwtSecret)
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	_, err = h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(auth.RefreshTokenExpiry),
	})
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	OK(w, map[string]interface{}{
		"access_token":  token,
		"refresh_token": refreshToken,
		"username":      user.Username,
		"user_id":       user.ID,
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		FailMsg(w, code.ErrInvalidArg, "缺少 refresh_token")
		return
	}

	rt, err := h.queries.GetRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		Fail(w, code.ErrUserRefreshTokenInvalid)
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), rt.UserID)
	if err != nil {
		Fail(w, code.ErrUserNotFound)
		return
	}

	accessToken, err := auth.GenerateToken(user.ID, user.Username, h.jwtSecret)
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	newRefreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	_ = h.queries.RevokeRefreshToken(r.Context(), req.RefreshToken)

	_, err = h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     newRefreshToken,
		ExpiresAt: time.Now().Add(auth.RefreshTokenExpiry),
	})
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	OK(w, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"username":      user.Username,
		"user_id":       user.ID,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		FailMsg(w, code.ErrInvalidArg, "缺少 refresh_token")
		return
	}

	_ = h.queries.RevokeRefreshToken(r.Context(), req.RefreshToken)

	OK(w, map[string]string{"message": "已退出登录"})
}

// GetMe 获取当前登录用户信息
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}
	claims := auth.GetClaims(r)
	if claims == nil {
		Fail(w, code.ErrUnauthorized)
		return
	}
	user, err := h.queries.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		Fail(w, code.ErrUserNotFound)
		return
	}
	OK(w, map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"level":      user.Level,
		"created_at": user.CreatedAt.Time.Format("2006-01-02 15:04:05"),
	})
}

// ListUsers 用户列表（需要 user:read 权限）
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}
	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}
	type UserResp struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Level     int32  `json:"level"`
		CreatedAt string `json:"created_at"`
	}
	resp := make([]UserResp, 0, len(users))
	for _, u := range users {
		resp = append(resp, UserResp{
			ID:        u.ID,
			Username:  u.Username,
			Level:     u.Level,
			CreatedAt: u.CreatedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}
	OK(w, resp)
}

// UpgradeUser 升级用户等级（需要 user:upgrade 权限）
func (h *Handler) UpgradeUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}
	var req struct {
		UserID int64 `json:"user_id"`
		Level  int32 `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.Level < 0 || req.Level > 3 {
		FailMsg(w, code.ErrInvalidArg, "等级必须在 0-3 之间")
		return
	}
	if err := h.queries.UpdateUserLevel(r.Context(), db.UpdateUserLevelParams{
		ID:    req.UserID,
		Level: req.Level,
	}); err != nil {
		Fail(w, code.ErrInternal)
		return
	}
	OK(w, map[string]interface{}{
		"user_id": req.UserID,
		"level":   req.Level,
	})
}

// ListWithdrawalLimits 查看所有限额配置（需要 limit:read 权限）
func (h *Handler) ListWithdrawalLimits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		Fail(w, code.ErrInvalidArg)
		return
	}
	limits, err := h.queries.ListWithdrawalLimits(r.Context())
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}
	OK(w, limits)
}

// UpdateWithdrawalLimit 修改限额配置（需要 limit:write 权限）
func (h *Handler) UpdateWithdrawalLimit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		Fail(w, code.ErrInvalidArg)
		return
	}
	var req struct {
		Level    int32  `json:"level"`
		BtcDaily string `json:"btc_daily"`
		EthDaily string `json:"eth_daily"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if err := h.queries.UpdateWithdrawalLimit(r.Context(), db.UpdateWithdrawalLimitParams{
		Level:    req.Level,
		BtcDaily: req.BtcDaily,
		EthDaily: req.EthDaily,
	}); err != nil {
		Fail(w, code.ErrInternal)
		return
	}
	OK(w, map[string]interface{}{
		"level":     req.Level,
		"btc_daily": req.BtcDaily,
		"eth_daily": req.EthDaily,
	})
}