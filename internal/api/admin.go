package api

import (
	"encoding/json"
	"net/http"

	"github.com/Ixecd/web3-blitz/internal/code"
	"github.com/Ixecd/web3-blitz/internal/db"
)

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
