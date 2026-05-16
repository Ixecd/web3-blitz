package api

import (
	"net/http"

	"github.com/Ixecd/blitz/internal/auth"
	"github.com/Ixecd/blitz/internal/db"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMux(h *Handler, jwtSecret string, queries *db.Queries) *http.ServeMux {
	mux := http.NewServeMux()

	// 公开接口（无需认证）
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// 登录/注册类端点 — 更严限速
	mux.HandleFunc("/api/v1/register", RateLimitMiddleware(h.authRL, h.Register))
	mux.HandleFunc("/api/v1/login", RateLimitMiddleware(h.authRL, h.Login))
	mux.HandleFunc("/api/v1/refresh", RateLimitMiddleware(h.authRL, h.Refresh))
	mux.HandleFunc("/api/v1/logout", RateLimitMiddleware(h.generalRL, h.Logout))
	mux.HandleFunc("/api/v1/forgot-password", RateLimitMiddleware(h.authRL, h.ForgotPassword))
	mux.HandleFunc("/api/v1/reset-password", RateLimitMiddleware(h.authRL, h.ResetPassword))

	// JWT 保护 + 全局限速（用户自身数据）
	mux.HandleFunc("/api/v1/address", auth.JWTMiddleware(jwtSecret,
		RateLimitMiddleware(h.generalRL, h.GenerateAddress)))
	mux.HandleFunc("/api/v1/balance", auth.JWTMiddleware(jwtSecret,
		RateLimitMiddleware(h.generalRL, h.GetBalance)))
	mux.HandleFunc("/api/v1/deposits", auth.JWTMiddleware(jwtSecret,
		RateLimitMiddleware(h.generalRL, h.ListDeposits)))
	mux.HandleFunc("/api/v1/balance/total", auth.JWTMiddleware(jwtSecret,
		RateLimitMiddleware(h.generalRL, h.GetTotalBalance)))
	mux.HandleFunc("/api/v1/withdrawals", auth.JWTMiddleware(jwtSecret,
		RateLimitMiddleware(h.generalRL, h.ListWithdrawals)))
	mux.HandleFunc("/api/v1/withdraw", auth.JWTMiddleware(jwtSecret,
		RateLimitMiddleware(h.generalRL, h.Withdraw)))
	mux.HandleFunc("/api/v1/users/me", auth.JWTMiddleware(jwtSecret,
		RateLimitMiddleware(h.generalRL, h.GetMe)))

	// JWT + RBAC 保护（管理员）
	mux.HandleFunc("/api/v1/users", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "user:read", h.ListUsers)))
	mux.HandleFunc("/api/v1/users/upgrade", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "user:upgrade", h.UpgradeUser)))
	mux.HandleFunc("/api/v1/withdrawal-limits", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "limit:read", h.ListWithdrawalLimits)))
	mux.HandleFunc("/api/v1/withdrawal-limits/update", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "limit:write", h.UpdateWithdrawalLimit)))
	// 管理员提币审核
	mux.HandleFunc("/api/v1/admin/withdrawals/pending", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "withdraw:review", RateLimitMiddleware(h.generalRL, h.ListPendingWithdrawals))))
	mux.HandleFunc("/api/v1/admin/withdrawals/approve", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "withdraw:review", RateLimitMiddleware(h.generalRL, h.ApproveWithdrawal))))
	mux.HandleFunc("/api/v1/admin/withdrawals/reject", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "withdraw:review", RateLimitMiddleware(h.generalRL, h.RejectWithdrawal))))

	return mux
}
