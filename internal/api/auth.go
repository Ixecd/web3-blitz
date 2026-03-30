package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Ixecd/web3-blitz/internal/auth"
	"github.com/Ixecd/web3-blitz/internal/code"
	"github.com/Ixecd/web3-blitz/internal/db"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.Email == "" || req.Password == "" {
		FailMsg(w, code.ErrInvalidArg, "email 和 password 不能为空")
		return
	}
	if len(req.Password) < 8 {
		FailMsg(w, code.ErrInvalidArg, "密码长度不能少于 8 位")
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	slog.Debug("注册请求", "email", req.Email)
	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Username: req.Email,
		Email:    req.Email,
		Password: hashed,
	})
	if err != nil {
		slog.Error("CreateUser 失败", "email", req.Email, "err", err)
		Fail(w, code.ErrInternal)
		return
	}
	slog.Debug("注册成功", "id", user.ID, "email", user.Email)

	OK(w, map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.Email == "" || req.Password == "" {
		FailMsg(w, code.ErrInvalidArg, "email 和 password 不能为空")
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), req.Email)
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
		"email":         user.Email,
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

// ForgotPassword POST /api/v1/forgot-password
// 无论邮箱是否存在都返回成功，避免邮箱枚举攻击。
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		Fail(w, code.ErrInvalidArg)
		return
	}

	// 查用户，不存在也返回 OK（防枚举）
	user, err := h.queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		OK(w, map[string]string{"message": "如果该邮箱已注册，重置链接将在几分钟内发送"})
		return
	}

	// 生成 32 字节随机 token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		Fail(w, code.ErrInternal)
		return
	}
	token := hex.EncodeToString(raw)

	_, err = h.queries.CreatePasswordResetToken(r.Context(), db.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	})
	if err != nil {
		slog.Error("CreatePasswordResetToken 失败", "user_id", user.ID, "err", err)
		Fail(w, code.ErrInternal)
		return
	}

	if err := h.mailer.SendResetEmail(user.Email, token); err != nil {
		slog.Error("发送重置邮件失败", "email", user.Email, "err", err)
		Fail(w, code.ErrInternal)
		return
	}

	OK(w, map[string]string{"message": "如果该邮箱已注册，重置链接将在几分钟内发送"})
}

// ResetPassword POST /api/v1/reset-password
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Fail(w, code.ErrInvalidArg)
		return
	}

	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, code.ErrInvalidArg)
		return
	}
	if req.Token == "" || len(req.Password) < 8 {
		FailMsg(w, code.ErrInvalidArg, "token 不能为空，密码长度不能少于 8 位")
		return
	}

	// 验证 token
	rt, err := h.queries.GetPasswordResetToken(r.Context(), req.Token)
	if err != nil {
		FailMsg(w, code.ErrInvalidArg, "重置链接已失效或不存在")
		return
	}

	// hash 新密码
	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	// 更新密码
	if err := h.queries.UpdateUserPassword(r.Context(), db.UpdateUserPasswordParams{
		ID:       rt.UserID,
		Password: hashed,
	}); err != nil {
		Fail(w, code.ErrInternal)
		return
	}

	// token 标记已使用
	_ = h.queries.MarkPasswordResetTokenUsed(r.Context(), req.Token)

	// 踢掉所有 refresh token，强制重新登录
	_ = h.queries.RevokeAllUserRefreshTokens(r.Context(), rt.UserID)

	OK(w, map[string]string{"message": "密码已重置，请重新登录"})
}
