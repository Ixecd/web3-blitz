package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Ixecd/web3-blitz/internal/auth"
	"github.com/Ixecd/web3-blitz/internal/db"
	"github.com/Ixecd/web3-blitz/internal/lock"
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

	var totalFloat float64
	if t, ok := total.(string); ok {
		fmt.Sscanf(t, "%f", &totalFloat)
	} else {
		totalFloat, _ = total.(float64)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"chain":   chainStr,
		"total":   totalFloat,
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

	// 分布式锁，防止同一用户同一链并发提币
	// lockKey := fmt.Sprintf("withdraw:lock:%s:%s", req.UserID, req.Chain)
	// locked, err := h.redis.SetNX(ctx, lockKey, 1, 30*time.Second).Result()
	// if err != nil {
	// 	http.Error(w, "获取锁失败: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// if !locked {
	// 	http.Error(w, "请勿重复提交，上一笔提币正在处理中", http.StatusTooManyRequests)
	// 	return
	// }
	// defer h.redis.Del(ctx, lockKey)
	lockKey := fmt.Sprintf("withdraw:%s:%s", req.UserID, req.Chain)
	l, err := h.locker.Acquire(ctx, lockKey)
	if err != nil {
		http.Error(w, "请勿重复提交: "+err.Error(), http.StatusTooManyRequests)
		return
	}
	defer l.Release(context.Background())

	// 已确认充值
	rawDeposit, err := h.queries.GetTotalDepositByUserIDAndChain(ctx, db.GetTotalDepositByUserIDAndChainParams{
		UserID: req.UserID,
		Chain:  string(req.Chain),
	})
	if err != nil {
		http.Error(w, "查询充值余额失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 已完成提币
	rawWithdrawal, err := h.queries.GetTotalWithdrawalByUserIDAndChain(ctx, db.GetTotalWithdrawalByUserIDAndChainParams{
		UserID: req.UserID,
		Chain:  string(req.Chain),
	})
	if err != nil {
		http.Error(w, "查询提币余额失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 类型转换（复用之前的模式）
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

	available := toFloat(rawDeposit) - toFloat(rawWithdrawal)
	if available < req.Amount {
		http.Error(w, fmt.Sprintf("余额不足: 可用 %.8f，请求 %.8f", available, req.Amount), http.StatusBadRequest)
		return
	}

	// 1. 写入 pending 记录，拿到 ID
	record, err := h.queries.CreateWithdrawal(ctx, db.CreateWithdrawalParams{
		UserID:  req.UserID,
		Address: req.ToAddress,
		Amount:  fmt.Sprintf("%.8f", req.Amount),
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
		Fee:    fmt.Sprintf("%.8f", fee),
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

func (h *Handler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "只支持 GET", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "缺少 user_id 参数", http.StatusBadRequest)
		return
	}

	withdrawals, err := h.queries.ListWithdrawalsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	for _, w := range withdrawals {
		var amount, fee float64
		fmt.Sscanf(w.Amount, "%f", &amount)
		fmt.Sscanf(w.Fee, "%f", &fee)
		resp = append(resp, WithdrawalResp{
			ID:        w.ID,
			TxID:      w.TxID.String,
			Address:   w.Address,
			UserID:    w.UserID,
			Amount:    amount,
			Fee:       fee,
			Status:    w.Status,
			Chain:     w.Chain,
			CreatedAt: w.CreatedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "参数错误", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username 和 password 不能为空", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "密码长度不能少于8位", http.StatusBadRequest)
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "密码加密失败", http.StatusInternalServerError)
		return
	}

	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Username: req.Username,
		Password: hashed,
	})
	if err != nil {
		http.Error(w, "用户名已存在", http.StatusConflict)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "参数错误", http.StatusBadRequest)
		return
	}

	user, err := h.queries.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		http.Error(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}

	if !auth.CheckPassword(req.Password, user.Password) {
		http.Error(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, h.jwtSecret)
	if err != nil {
		http.Error(w, "生成 token 失败", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":    token,
		"username": user.Username,
		"user_id":  user.ID,
	})
}
